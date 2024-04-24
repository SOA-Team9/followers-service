package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"followers-service.xws.com/handler"
	"followers-service.xws.com/repo"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	//Reading from environment, if not set we will default it to 8080.
	//This allows flexibility in different environments (for eg. when running multiple docker api's and want to override the default port)
	/*
		err := godotenv.Load(".env")

		if err != nil {
			log.Fatalf("Error loading .env file")
		}
	*/
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8086"
	}

	// Initialize context
	timeoutContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//Initialize the logger we are going to use, with prefix and datetime for every log
	logger := log.New(os.Stdout, "[followers-api] ", log.LstdFlags)
	followLogger := log.New(os.Stdout, "[follow-store] ", log.LstdFlags)

	// NoSQL: Initialize Repository stores
	fstore, err := repo.NewFollowsStore(followLogger)
	if err != nil {
		logger.Fatal(err)
	}
	defer fstore.CloseDriverConnection(timeoutContext)
	fstore.CheckConnection()
	//------------------------------------------------------------
	followLogger.Println("I AM IN MAIN")

	//Initialize the handlers and inject said logger
	//moviesHandler := handlers.NewMoviesHandler(logger, store)
	followsHandler := handler.NewFollowsHandler(followLogger, fstore)

	//Initialize the router and add a middleware for all the requests
	router := mux.NewRouter()

	//Follows API

	// Define subrouter for POST /user
	router.Handle("/user", followsHandler.MiddlewareContentTypeSet(followsHandler.MiddlewareUserDeserialization(http.HandlerFunc(followsHandler.AddUser)))).Methods(http.MethodPost)

	// Define subrouter for POST /follows
	router.Handle("/follows", followsHandler.MiddlewareContentTypeSet(followsHandler.MiddlewareFollowDeserialization(http.HandlerFunc(followsHandler.FollowUser)))).Methods(http.MethodPost)

	// Define subrouter for GET /check-following
	router.Handle("/check-following", followsHandler.MiddlewareContentTypeSet(followsHandler.MiddlewareFollowDeserialization(http.HandlerFunc(followsHandler.CheckFollow)))).Methods(http.MethodGet)
	router.Handle("/unfollow/{followedId}/{followingId}", followsHandler.MiddlewareContentTypeSet(http.HandlerFunc(followsHandler.UnfollowUser))).Methods(http.MethodDelete)

	// Define subrouter for GET /user/{user_id}/following
	router.Handle("/user/following/{user_id}", followsHandler.MiddlewareContentTypeSet(http.HandlerFunc(followsHandler.GetUserFollowing))).Methods(http.MethodGet)
	router.Handle("/user/followers/{user_id}", followsHandler.MiddlewareContentTypeSet(http.HandlerFunc(followsHandler.GetUserFollowers))).Methods(http.MethodGet)
	router.Handle("/user/following-ids/{user_id}", followsHandler.MiddlewareContentTypeSet(http.HandlerFunc(followsHandler.GetUserFollowingIds))).Methods(http.MethodGet)
	router.Handle("/user/{user_id}", followsHandler.MiddlewareContentTypeSet(http.HandlerFunc(followsHandler.GetUserFollowing))).Methods(http.MethodGet)

	router.Handle("/recommendation/{user_id}", followsHandler.MiddlewareContentTypeSet(http.HandlerFunc(followsHandler.GetFollowingRecommendation))).Methods(http.MethodGet)

	router.Handle("/test", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		followLogger.Println("I AM IN TEST")
		rw.WriteHeader(http.StatusOK)
	})).Methods(http.MethodGet)

	//CORS
	cors := gorillaHandlers.CORS(gorillaHandlers.AllowedOrigins([]string{"*"}))

	//Initialize the server
	server := http.Server{
		Addr:         ":" + port,
		Handler:      cors(router),
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	logger.Println("Server listening on port", port)

	serverError := server.ListenAndServe()

	if serverError != nil {
		logger.Fatal(serverError)
	}
}

/*
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

var err error

func main() {
	//port := os.Getenv("PORT")

	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	//Initialize the logger we are going to use, with prefix and datetime for every log
	logger := log.New(os.Stdout, "[followers-api] ", log.LstdFlags)
	followLogger := log.New(os.Stdout, "[follow-store] ", log.LstdFlags)

	// NoSQL: Initialize Repository stores
	fstore, err := repo.NewFollowsStore(followLogger)
	if err != nil {
		logger.Fatal(err)
	}
	defer fstore.CloseDriverConnection(timeoutContext)
	fstore.CheckConnection()

	handler.NewFollowsHandler(followLogger, fstore)

	router := gin.Default()

	router.POST("/user", func(ctx *gin.Context) {
		fmt.Println("I am in /user")
	})
	router.POST("/follows", func(ctx *gin.Context) {
		fmt.Println("I am in /follows")
	})
	router.POST("/check-following", func(ctx *gin.Context) {
		fmt.Println("I am in /check-following")
	})

	router.GET("/following/:userId", func(ctx *gin.Context) {
		fmt.Println("I am in /following/{userId}")
	})

	router.GET("/recommendation/:userId", func(ctx *gin.Context) {
		fmt.Println("I am in /recommendation/{userId}")
	})

	fmt.Println("Server listening on port", "8086")
	http.ListenAndServe(":8086", cors.AllowAll().Handler(router))
}*/
