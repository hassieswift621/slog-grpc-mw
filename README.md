# slog-grpc-mw
[![Godoc reference](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://pkg.go.dev/github.com/hassieswift621/slog-grpc-mw@v1.0.0?tab=doc)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/hassieswift621/slog-grpc-mw?logo=go&style=flat-square)](https://github.com/hassieswift621/slog-grpc-mw/releases)

gRPC Go middleware for Cdr's Slog logger: <https://github.com/cdr/slog>

![Output Screenshot](https://raw.githubusercontent.com/hassieswift621/slog-grpc-mw/dev/screenshot_output.png)

## Install
```text
go get github.com/hassieswift621/slog-grpc-mw
```

## Example Usage
```go
func main() {
    // Create GRPC server.
    srv := grpc.NewServer(
        grpc.UnaryInterceptor(
            grpc_middleware.ChainUnaryServer(
                grpc_slog.UnaryServerInterceptor(grpcLogger(true)),
            ),
        ),
    )
}

func grpcLogger(verbose bool) slog.Logger {
	logger := sloghuman.Make(os.Stdout).Leveled(slog.LevelWarn).Named("grpc")

	if verbose {
		logger = logger.Leveled(slog.LevelDebug)
	}

	return logger
}
```

## License
```text
Copyright ©2020 Hassie.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```