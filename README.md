# Library

В этом домашнем задании вам предстоит реализовать свой собственный сервис **library**.
В последующих домашних заданиях вы будете его развивать

## API

### Валидация
* ID книги и автора должны быть в формате [UUID](https://ru.wikipedia.org/wiki/UUID)
* author_name должно допускаться регулярным выражением `^[A-Za-z0-9]+( [A-Za-z0-9]+)*$`, при этом len(author_name) in [1; 512]
* Приветствуется дополнительная валидация, если она не противоречит тестам и здравому смыслу

### REST to gRPC
Должна быть поддержка REST to gRPC API с соответствующими путями и типами запросов, указанных ниже

```protobuf
syntax = "proto3";

import "google/api/annotations.proto";
import "validate/validate.proto";

package library;

option go_package = "github.com/project/library/pkg/api/library;library";

service Library {
  // post: "/v1/library/book"
  rpc AddBook(AddBookRequest) returns (AddBookResponse) {}
  
  // put: "/v1/library/book"
  rpc UpdateBook(UpdateBookRequest) returns (UpdateBookResponse) {}

  // get: "/v1/library/book/{id}"
  rpc GetBookInfo(GetBookInfoRequest) returns (GetBookInfoResponse) {}

  // post: "/v1/library/author"
  rpc RegisterAuthor(RegisterAuthorRequest) returns (RegisterAuthorResponse) {}

  // put: "/v1/library/author"
  rpc ChangeAuthorInfo(ChangeAuthorInfoRequest) returns (ChangeAuthorInfoResponse) {}

  // get: "/v1/library/author/{id}"
  rpc GetAuthorInfo(GetAuthorInfoRequest) returns (GetAuthorInfoResponse) {}

  // get: "/v1/library/author_books/{author_id}"
  rpc GetAuthorBooks(GetAuthorBooksRequest) returns (stream Book) {}
}

message Book {
  string id = 1;
  string name = 2;
  repeated string author_id = 3;
}

message AddBookRequest {
  string name = 1;
  repeated string author_ids = 2;
}

message AddBookResponse {
  Book book = 1;
}

message UpdateBookRequest {
  string id = 1;
  string name = 2;
  repeated string author_ids = 3;
}

message UpdateBookResponse {}

message GetBookInfoRequest {
  string id = 1;
}

message GetBookInfoResponse {
  Book book = 1;
}

message RegisterAuthorRequest {
  string name = 1;
}

message RegisterAuthorResponse {
  string id = 1;
}

message ChangeAuthorInfoRequest {
  string id = 1;
  string name = 2;
}

message ChangeAuthorInfoResponse {}

message GetAuthorInfoRequest {
  string id = 1;
}

message GetAuthorInfoResponse {
  string id = 1;
  string name = 2;
}

message GetAuthorBooksRequest {
  string author_id = 1;
}
```


## Унификация технологий
Для удобства выполнения и проверки дз вводится ряд правил, унифицирующих используемые технологии

* Структура проекта [go-clean-template](https://github.com/evrone/go-clean-template) и этот [шаблон](https://github.com/itmo-org/lectures/tree/main/sem2/lecture1)
* Для генерации кода авторские [Makefile](./Makefile) и [easyp.yaml](./easyp.yaml).
* Для логирования [zap](https://github.com/uber-go/zap)
* Для валидации [protoc-gen-validate](https://github.com/bufbuild/protoc-gen-validate)
* Для поддержики REST-to-gRPC API [gRPC gateway](https://grpc-ecosystem.github.io/grpc-gateway/)

## Тестирование в CI
* Код тестов можно посмотреть в файле [integration_test.go](./integration-test/integration_test.go).
* Важно, чтобы ваш сервис умел корректно обрабатывать SIGINT и SIGTERM, иначе тесты могут работать некорректно.
* В [Makefile](Makefile) реализованы метки **build** и **generate**, без них CI не будет работать

## Переменные окружения
В рамках вашего сервиса вы должны реализовать конфиг, который будет работать с переменными окружения

* GRPC_PORT порт для gRPC сервера
* GRPC_GATEWAY_PORT порт для REST to gRPC API (gRPC gateway)

## Тесты
Необходимо сгенерировать моки и написать свои тесты, степень покрытия будет проверяться в CI

## Документация
Вам необходимо своими словами написать [README.md](./docs/README.md) в ./docs к своему сервису library

## Рекомендации
* [Пример реализации](https://github.com/itmo-org/lectures/tree/main/sem2/lecture1)
* Не забывайте про логирование
* После генерации swagger.json, вы можете посмотреть на REST API вашего сервиса в [swagger editor](https://editor.swagger.io/).

## Особенности реализации
- Используйте [тесты](./integration-test), чтобы осознать недосказанности.
- В данном домашнем задании необходимо реализовать in-memory хранилище, которое потом будет заменено на базу данных.

## Письменные комментарии
Поскольку количество попыток сдачи ограничено, вы можете написать дополнительные комментарии в PR. Если ваше
обоснование будет достаточно разумным, это может быть учтено при выставлении баллов. Например,

* описать, почему вы написали именно такие интерфейсы
* описать, почему вы сделали именно такую валидацию
* описать, почему вы сделали именно такую схему в базе данных

## Сдача
* Открыть pull request из ветки `hw` в ветку `main` **вашего репозитория**.
* В описании PR заполнить количество часов, которые вы потратили на это задание.
* Отправить заявку на ревью в соответствующей форме.
* Время дедлайна фиксируется отправкой формы.
* Изменять файлы в ветке main без PR запрещено.
* Изменять файл [CI workflow](./.github/workflows/library.yaml) запрещено.

## Makefile
Для удобств локальной разработки сделан [`Makefile`](Makefile). Имеются следующие команды:

Запустить полный цикл (линтер, тесты):

```bash 
make all
```

Запустить только тесты:

```bash
make test
``` 

Запустить линтер:

```bash
make lint
```

Подтянуть новые тесты:

```bash
make update
```

При разработке на Windows рекомендуется использовать [WSL](https://learn.microsoft.com/en-us/windows/wsl/install), чтобы
была возможность пользоваться вспомогательными скриптами.
