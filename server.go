//go:generate go run github.com/99designs/gqlgen generate

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/cccac/OpenWarn-Proxy/graph"
	"github.com/cccac/OpenWarn-Proxy/graph/generated"
	"github.com/cccac/OpenWarn-Proxy/proxy"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	proxy := proxy.New()
	go proxy.UpdateLoop()

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{
		Proxy: &proxy,
	}}))

	http.Handle("/", playground.Handler("OpenWarn-Proxy playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
