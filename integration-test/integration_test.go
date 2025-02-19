//go:build integration_test

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestLibraryWithInMemoryInvariant(t *testing.T) {
	executable := getLibraryExecutable(t)
	grpcPort := findFreePort(t)
	grpcGatewayPort := findFreePort(t)

	http.DefaultClient.Timeout = time.Second * 1

	cmd := setupLibrary(t, executable, grpcPort, grpcGatewayPort)
	t.Cleanup(func() {
		stopLibrary(t, cmd)
	})

	t.Run("author grpc", func(t *testing.T) {
		ctx := context.Background()
		client := newGRPCClient(t, grpcPort)

		const authorName = "Test testovich"

		registerRes, err := client.RegisterAuthor(ctx, &RegisterAuthorRequest{
			Name: authorName,
		})

		require.NoError(t, err)
		authorID := registerRes.GetId()

		author, err := client.GetAuthorInfo(ctx, &GetAuthorInfoRequest{
			Id: authorID,
		})
		require.NoError(t, err)

		require.Equal(t, authorName, author.GetName())
		require.Equal(t, authorID, author.GetId())

		_, err = client.ChangeAuthorInfo(ctx, &ChangeAuthorInfoRequest{
			Id:   authorID,
			Name: authorName + "123",
		})
		require.NoError(t, err)

		newAuthor, err := client.GetAuthorInfo(ctx, &GetAuthorInfoRequest{
			Id: authorID,
		})
		require.NoError(t, err)

		require.Equal(t, authorName+"123", newAuthor.GetName())
		require.Equal(t, authorID, newAuthor.GetId())
	})

	t.Run("book grpc", func(t *testing.T) {
		ctx := context.Background()
		client := newGRPCClient(t, grpcPort)

		const (
			authorName = "Test testovich"
			bookName   = "go"
		)

		registerRes, err := client.RegisterAuthor(ctx, &RegisterAuthorRequest{
			Name: authorName,
		})

		require.NoError(t, err)
		authorID := registerRes.GetId()

		response, err := client.AddBook(ctx, &AddBookRequest{
			Name:     bookName,
			AuthorId: []string{authorID},
		})
		require.NoError(t, err)

		book := response.GetBook()

		require.Equal(t, bookName, book.GetName())
		require.Equal(t, 1, len(book.GetAuthorId()))
		require.Equal(t, authorID, book.GetAuthorId()[0])

		_, err = client.UpdateBook(ctx, &UpdateBookRequest{
			Id:        book.GetId(),
			Name:      bookName + "-2024",
			AuthorIds: []string{authorID},
		})
		require.NoError(t, err)

		newBook, err := client.GetBookInfo(ctx, &GetBookInfoRequest{
			Id: book.GetId(),
		})

		require.NoError(t, err)
		require.Equal(t, bookName+"-2024", newBook.GetBook().GetName())
		require.Equal(t, 1, len(newBook.GetBook().GetAuthorId()))
		require.Equal(t, authorID, newBook.GetBook().GetAuthorId()[0])

		books := getAllAuthorBooks(t, authorID, client)

		require.NoError(t, err)
		require.Equal(t, 1, len(books))

		require.Equal(t, newBook.GetBook().GetName(), books[0].GetName())
		require.Equal(t, newBook.GetBook().GetAuthorId(), books[0].GetAuthorId())
	})

	t.Run("concurrent access", func(t *testing.T) {
		ctx := context.Background()
		client := newGRPCClient(t, grpcPort)

		var (
			authorName = "Test testovich" + strconv.Itoa(rand.N[int](10e9))
			totalBooks = 1234
			workers    = 50
		)

		registerRes, err := client.RegisterAuthor(ctx, &RegisterAuthorRequest{
			Name: authorName,
		})

		require.NoError(t, err)
		authorID := registerRes.GetId()

		books := make([]string, 0, totalBooks)
		for i := range totalBooks {
			books = append(books, strconv.Itoa(i))
		}

		perWorker := totalBooks / workers
		start := 0

		wg := new(sync.WaitGroup)
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(s int) {
				defer wg.Done()

				right := s + perWorker
				if i == workers-1 {
					right = len(books)
				}

				for b := s; b < right; b++ {
					_, err := client.AddBook(ctx, &AddBookRequest{
						Name:     books[b],
						AuthorId: []string{authorID},
					})
					require.NoError(t, err)
				}
			}(start)

			start += perWorker
		}

		wg.Wait()

		authorBooks := lo.Map(getAllAuthorBooks(t, authorID, client), func(item *Book, index int) string {
			return item.GetName()
		})

		slices.Sort(authorBooks)
		slices.Sort(books)

		require.Equal(t, books, authorBooks)
	})

	t.Run("author not found", func(t *testing.T) {
		ctx := context.Background()
		client := newGRPCClient(t, grpcPort)

		_, err := client.GetAuthorInfo(ctx, &GetAuthorInfoRequest{
			Id: uuid.New().String(),
		})

		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("author invalid argument", func(t *testing.T) {
		ctx := context.Background()
		client := newGRPCClient(t, grpcPort)

		_, err := client.GetAuthorInfo(ctx, &GetAuthorInfoRequest{
			Id: "123",
		})

		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("book not found", func(t *testing.T) {
		ctx := context.Background()
		client := newGRPCClient(t, grpcPort)

		_, err := client.GetBookInfo(ctx, &GetBookInfoRequest{
			Id: uuid.New().String(),
		})

		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("book invalid argument", func(t *testing.T) {
		ctx := context.Background()
		client := newGRPCClient(t, grpcPort)

		_, err := client.GetBookInfo(ctx, &GetBookInfoRequest{
			Id: "123",
		})

		s, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("grpc gateway", func(t *testing.T) {
		type RegisterAuthorResponse struct {
			ID string `json:"id"`
		}

		type GetAuthorResponse struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}

		registerUrl := fmt.Sprintf("http://127.0.0.1:%s/v1/library/author", grpcGatewayPort)

		request, err := http.NewRequest("POST", registerUrl, strings.NewReader(`{"name": "Name"}`))
		require.NoError(t, err)

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)

		data, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		var registerAuthorResponse RegisterAuthorResponse

		err = json.Unmarshal(data, &registerAuthorResponse)
		require.NoError(t, err)

		require.NotEmpty(t, registerAuthorResponse)

		getUrl := fmt.Sprintf("http://127.0.0.1:%s/v1/library/author/%s",
			grpcGatewayPort, registerAuthorResponse.ID)

		getRequest, err := http.NewRequest("GET", getUrl, nil)
		require.NoError(t, err)

		getResponse, err := http.DefaultClient.Do(getRequest)
		require.NoError(t, err)

		getData, err := io.ReadAll(getResponse.Body)
		require.NoError(t, err)

		var author GetAuthorResponse
		err = json.Unmarshal(getData, &author)
		require.NoError(t, err)

		require.Equal(t, author.ID, registerAuthorResponse.ID)
		require.Equal(t, author.Name, "Name")
	})

	t.Run("book many authors grpc", func(t *testing.T) {
		ctx := context.Background()
		client := newGRPCClient(t, grpcPort)

		var (
			authorBasicName = "Donald Knuth"
			authorsCount    = 10
			bookName        = "The Art of Computer Programming"
		)

		authorIds := make([]string, authorsCount)
		for i := range authorsCount {
			author, err := client.RegisterAuthor(ctx, &RegisterAuthorRequest{
				Name: authorBasicName + strconv.Itoa(rand.N[int](10e9)),
			})
			require.NoError(t, err)
			authorIds[i] = author.Id
		}

		bookAdded, err := client.AddBook(ctx, &AddBookRequest{
			Name:     bookName,
			AuthorId: authorIds,
		})
		require.NoError(t, err)
		require.ElementsMatch(t, bookAdded.Book.AuthorId, authorIds)

		bookReceived, err := client.GetBookInfo(ctx, &GetBookInfoRequest{
			Id: bookAdded.Book.Id,
		})
		require.NoError(t, err)
		require.EqualExportedValues(t, bookAdded.Book, bookReceived.Book)
	})

	t.Run("grpc gateway unknown url", func(t *testing.T) {
		unknownUrl := fmt.Sprintf("http://127.0.0.1:%s/v0/not_library/not_author", grpcGatewayPort)

		response, err := http.Get(unknownUrl)

		require.NoError(t, err)
		require.Equal(t, response.StatusCode, 404)
	})
}

func getLibraryExecutable(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	binaryPath, err := resolveFilePath(filepath.Dir(wd), "library")
	require.NoError(t, err, "you need to compile your library service, run make build")

	return binaryPath
}

var requiredEnv = []string{"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD"}

func setupLibrary(
	t *testing.T,
	executable string,
	grpcPort string,
	grpcGatewayPort string,
) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(executable)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	for _, p := range requiredEnv {
		cur := os.Getenv(p)
		require.NotEmpty(t, cur, "you need to pass env variable to tests: "+p)

		cmd.Env = append(cmd.Env, p+"="+cur)
	}

	cmd.Env = append(cmd.Env, "GRPC_PORT="+grpcPort)
	cmd.Env = append(cmd.Env, "GRPC_GATEWAY_PORT="+grpcGatewayPort)

	require.NoError(t, cmd.Start())
	grpcClient := newGRPCClient(t, grpcPort)

	// grpc health check
	for i := range 50 {
		// use idempotent request and validation for healthcheck
		_, err := grpcClient.GetBookInfo(context.Background(), &GetBookInfoRequest{
			Id: "123", // not invalid
		})

		_, ok := status.FromError(err)

		if ok {
			break
		}

		if i == 19 {
			log.Println("grpc health check error")
			t.Fail()
		}

		time.Sleep(time.Millisecond * 100)
	}

	// gateway health check
	unknownUrl := fmt.Sprintf("http://127.0.0.1:%s/v0/not_library/not_author_info", grpcGatewayPort)
	for i := range 50 {
		response, _ := http.Get(unknownUrl)

		if response != nil && response.StatusCode == http.StatusNotFound {
			break
		}

		if i == 19 {
			log.Println("gateway health check error")
			t.Fail()
		}

		time.Sleep(time.Millisecond * 10)
	}

	return cmd
}

func stopLibrary(t *testing.T, cmd *exec.Cmd) {
	t.Helper()

	for i := 0; i < 5; i++ {
		require.NoError(t, cmd.Process.Signal(syscall.SIGTERM))
	}

	require.NoError(t, cmd.Wait())
	require.Equal(t, 0, cmd.ProcessState.ExitCode())
}

func findFreePort(t *testing.T) string {
	t.Helper()

	for {
		port := rand.N(16383) + 49152
		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)

		if err == nil {
			require.NoError(t, ln.Close())
			return strconv.Itoa(port)
		}
	}
}

func newGRPCClient(t *testing.T, grpcPort string) LibraryClient {
	t.Helper()

	addr := "127.0.0.1:" + grpcPort
	c, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return NewLibraryClient(c)
}

func getAllAuthorBooks(t *testing.T, authorID string, client LibraryClient) []*Book {
	t.Helper()
	ctx := context.Background()

	result := make([]*Book, 0)
	stream, err := client.GetAuthorBooks(ctx, &GetAuthorBooksRequest{
		AuthorId: authorID,
	})
	require.NoError(t, err)

	for {
		resp, err := stream.Recv()

		if err == io.EOF {
			return result
		}

		require.NoError(t, err)

		result = append(result, resp)
	}
}

func resolveFilePath(root string, filename string) (string, error) {
	cleanedRoot := filepath.Clean(root)
	nameWithoutExt := strings.TrimRight(root, filepath.Ext(filename))

	var result string

	err := filepath.WalkDir(cleanedRoot, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		name := d.Name()

		if name == filename || name == nameWithoutExt {
			result = path
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("walk fail tree fail, error: %w", err)
	}

	if result == "" {
		return "", fmt.Errorf("file %s not found in root %s", filename, root)
	}

	return result, nil
}
