package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/ospatil/markd/internal/app"
	"github.com/ospatil/markd/internal/store"
)

var chiLambda *chiadapter.ChiLambdaV2

func init() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/tmp/markd.db"
	}

	db, err := store.New(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	chiLambda = chiadapter.NewV2(app.NewRouter(db))
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return chiLambda.ProxyWithContextV2(ctx, req)
}

func main() {
	lambda.Start(handler)
}
