package main

import (
	"api/internal/routes"
	"api/internal/types"
	"context"
	"log"
	"net/http"
	"os/signal"
	"shared/kafka"
	"shared/mongodb"
	"shared/utils"
	"strings"
	"syscall"
	"time"

	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	corsConfig := cors.DefaultConfig()

	corsAllowedOrigins := os.Getenv("CORS_ALLOW_ORIGINS")
	// Split the origins into a slice, assuming they are comma-separated
	allowedOrigins := strings.Split(corsAllowedOrigins, ",")

	// If no origins are specified, error out
	if len(allowedOrigins) == 0 {
		log.Fatal("CORS: No allowed origins specified, please specify with CORS_ALLOW_ORIGINS environment variable")
	}

	corsConfig.AllowOrigins = allowedOrigins
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	corsConfig.AllowCredentials = true
	r.Use(cors.New(corsConfig))

	mongoService, cleanup, err := mongodb.NewService()
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	producer, err := kafka.CreateProducer()
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	// Setup routes
	params := types.RouteParams{
		MongoService:  mongoService,
		KafkaProducer: producer,
	}
	routes.SetupRoutes(r, &params)

	// Handle 404s
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
	})

	// Check if running in AWS Lambda or not
	if utils.RunningInAWSLambda() {
		// Create a Lambda server with the Gin router
		// This converts lambda events to http.Request objects
		ginLambda := ginadapter.New(r)
		lambda.Start(func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			// Proxy the request to the Gin engine
			return ginLambda.ProxyWithContext(ctx, req)
		})
	} else {
		// Running outside AWS Lambda
		// This is to handle graceful shutdown (will close connections to MongoDB with the defer cleanup)
		srv := &http.Server{
			Addr:    ":8080",
			Handler: r,
		}

		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
			}
		}()

		// Wait for interrupt signal to gracefully shut down the server
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		// The context is used to inform the server it has 5 seconds to finish
		// the request it is currently handling
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal("Server forced to shutdown:", err)
		}

		log.Println("Server exiting")
	}
}