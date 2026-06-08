// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/Tanmoy095/LogiSynapse/services/graphql-gateway/client"          // gRPC client
	"github.com/Tanmoy095/LogiSynapse/services/graphql-gateway/graph"           // GraphQL resolvers
	"github.com/Tanmoy095/LogiSynapse/services/graphql-gateway/graph/generated" // Generated GraphQL schema
)

// main starts the GraphQL server and connects to the Shipment Service.
// Analogy: Opens the restaurant, sets up the waiter's intercom, and starts serving customers.
func main() {
	addr := os.Getenv("SHIPMENT_SERVICE_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}
	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authAddr == "" {
		authAddr = "localhost:50052"
	}

	// Initialize gRPC client to connect to Shipment Service
	// Analogy: Set up the waiter's intercom to call the kitchen
	shipmentClient, err := client.NewShipmentClient(addr)
	if err != nil {
		log.Fatalf("failed to connect to shipment service: %v", err)
	}
	defer shipmentClient.Close() // Close connection when server stops

	authClient, err := client.NewAuthClient(authAddr)
	if err != nil {
		log.Fatalf("failed to connect to authentication service: %v", err)
	}
	defer authClient.Close()

	// Initialize GraphQL resolver with gRPC client
	resolver := graph.NewResolver(shipmentClient, authClient)

	// Set up health check endpoint (no auth required)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"graphql-gateway"}`))
	})

	// Set up GraphQL endpoint at /query
	// Analogy: Set up the dining room's service counter for customer orders
	http.Handle("/query", authMiddleware(authClient, handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))))

	// Set up GraphiQL playground at root (/) for easy testing
	// Analogy: Provide a menu board for customers to write their orders
	http.Handle("/", playground.Handler("GraphQL Playground", "/query"))

	log.Println("GraphQL server running on :8080")
	log.Println("  - GraphQL endpoint: http://localhost:8080/query")
	log.Println("  - GraphiQL playground: http://localhost:8080/")
	log.Println("  - Health check: http://localhost:8080/health")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func authMiddleware(authClient *client.AuthClient, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow GET requests (GraphiQL playground) without authentication
		if r.Method == http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		
		// Require authentication for POST requests (GraphQL mutations/queries)
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing Authorization bearer token", http.StatusUnauthorized)
			return
		}
		
		tenantID := r.Header.Get("X-Tenant-ID")
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		
		identity, err := authClient.ValidateAccessToken(ctx, authHeader, tenantID)
		if err != nil || !identity.Allowed {
			http.Error(w, "invalid or unauthorized token", http.StatusUnauthorized)
			return
		}
		
		// Store identity in request context for resolvers to use
		next.ServeHTTP(w, r.WithContext(graph.WithIdentity(r.Context(), graph.Identity{
			UserID:      identity.UserID,
			Email:       identity.Email,
			TenantID:    identity.TenantID,
			Role:        identity.Role,
			IsSuperAdmin: identity.IsSuperAdmin,
		})))
	})
}
