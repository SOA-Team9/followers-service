package data

import (
	"context"
	"log"
	"os"

	// NoSQL: module containing Neo4J api client
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// NoSQL: MovieRepo struct encapsulating Neo4J api client
type MovieRepo struct {
	// Thread-safe instance which maintains a database connection pool
	driver neo4j.DriverWithContext
	logger *log.Logger
}

// NoSQL: Constructor which reads db configuration from environment and creates a keyspace
func New(logger *log.Logger) (*MovieRepo, error) {
	// Local instance
	uri := os.Getenv("NEO4J_DB")
	user := os.Getenv("NEO4J_USERNAME")
	pass := os.Getenv("NEO4J_PASS")
	auth := neo4j.BasicAuth(user, pass, "")

	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		logger.Panic(err)
		return nil, err
	}

	// Return repository with logger and DB session
	return &MovieRepo{
		driver: driver,
		logger: logger,
	}, nil
}

// Check if connection is established
func (mr *MovieRepo) CheckConnection() {
	ctx := context.Background()
	err := mr.driver.VerifyConnectivity(ctx)
	if err != nil {
		mr.logger.Panic(err)
		return
	}
	// Print Neo4J server address
	mr.logger.Printf(`Neo4J server address: %s`, mr.driver.Target().Host)
}

// Disconnect from database
func (mr *MovieRepo) CloseDriverConnection(ctx context.Context) {
	mr.driver.Close(ctx)
}

func (mr *MovieRepo) WritePerson(person *Person) error {
	// Neo4J Sessions are lightweight so we create one for each transaction (Cassandra sessions are not lightweight!)
	// Sessions are NOT thread safe
	ctx := context.Background()
	session := mr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	// ExecuteWrite for write transactions (Create/Update/Delete)
	savedPerson, err := session.ExecuteWrite(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				"CREATE (p:Person) SET p.born = $born, p.name = $name RETURN p.name + ', from node ' + id(p)",
				map[string]any{"born": person.Born, "name": person.Name})
			if err != nil {
				return nil, err
			}

			if result.Next(ctx) {
				return result.Record().Values[0], nil
			}

			return nil, result.Err()
		})
	if err != nil {
		mr.logger.Println("Error inserting Person:", err)
		return err
	}
	mr.logger.Println(savedPerson.(string))
	return nil
}

func (mr *MovieRepo) GetAllNodesWithMovieLabel(limit int) (Movies, error) {
	ctx := context.Background()
	session := mr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	// ExecuteRead for read transactions (Read and queries)
	movieResults, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				`MATCH (movie:Movie)
				RETURN movie.title as title, movie.released as released, movie.tagline as tagline
				LIMIT $limit`,
				map[string]any{"limit": limit})
			if err != nil {
				return nil, err
			}

			// Option 1: we iterate over result while there are records
			var movies Movies
			for result.Next(ctx) {
				record := result.Record()
				released, ok := record.Get("released")
				if !ok || released == nil {
					released = 0
				}
				title, _ := record.Get("title")
				tagline, _ := record.Get("tagline")
				movies = append(movies, &Movie{
					Released: released.(int64),
					Title:    title.(string),
					Tagline:  tagline.(string),
				})
			}
			return movies, nil
			// Option 2: we collect all records from result and iterate and map outside of the transaction
			// return result.Collect(ctx)
		})
	if err != nil {
		mr.logger.Println("Error querying search:", err)
		return nil, err
	}
	return movieResults.(Movies), nil
}

func (mr *MovieRepo) GetAllNodesWithMovieLabelAndGivenTitle(title string) (Movies, error) {
	ctx := context.Background()
	session := mr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	// ExecuteRead for read transactions (Read and queries)
	movieResults, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				`MATCH (movie:Movie) 
				 WHERE TOLOWER(movie.title) CONTAINS TOLOWER($title)
				 RETURN movie.title as title, movie.tagline as tagline, movie.released as released`,
				map[string]interface{}{"title": title})
			if err != nil {
				return nil, err
			}

			// Option 1: we iterate over result while there are records
			var movies Movies
			for result.Next(ctx) {
				record := result.Record()
				released, ok := record.Get("released")
				if !ok || released == nil {
					released = 0
				}
				title, _ := record.Get("title")
				tagline, _ := record.Get("tagline")
				movies = append(movies, &Movie{
					Released: released.(int64),
					Title:    title.(string),
					Tagline:  tagline.(string),
				})
			}
			return movies, nil
			// Option 2: we collect all records from result and iterate and map outside of the transaction
			// return result.Collect(ctx)
		})
	if err != nil {
		mr.logger.Println("Error querying search:", err)
		return nil, err
	}
	return movieResults.(Movies), nil
}

func (mr *MovieRepo) GetAllMoviesWithCast(limit int) (Movies, error) {
	ctx := context.Background()
	session := mr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	// ExecuteRead for read transactions (Read and queries)
	movieResults, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				`MATCH (m:Movie)<-[:ACTED_IN]-(p:Person)
				RETURN m.title as title, collect(p.name) as cast
				LIMIT $limit`,
				map[string]any{"limit": limit})
			if err != nil {
				return nil, err
			}

			// Option 1: we iterate over result while there are records
			var movies Movies
			for result.Next(ctx) {
				record := result.Record()
				title, _ := record.Get("title")
				cast, _ := record.Get("cast")
				movies = append(movies, &Movie{
					Title: title.(string),
					Cast:  mr.convertDataToPersonSlice(cast),
				})
			}
			return movies, nil
			// Option 2: we collect all records from result and iterate and map outside of the transaction
			// return result.Collect(ctx)
		})
	if err != nil {
		mr.logger.Println("Error querying search:", err)
		return nil, err
	}
	return movieResults.(Movies), nil
}

func (mr *MovieRepo) GetPersonWhoPlayedRole(role string) (Roles, error) {
	ctx := context.Background()
	session := mr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	// ExecuteRead for read transactions (Read and queries)
	roleResults, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				`MATCH (m:Movie)<-[r:ACTED_IN]-(p:Person)
				WHERE $role in r.roles 
				RETURN m.title as title, p.name as actor, r.roles[0] as role`,
				map[string]any{"role": role})
			if err != nil {
				return nil, err
			}

			// Option 1: we iterate over result while there are records
			var roles Roles
			for result.Next(ctx) {
				record := result.Record()
				title, _ := record.Get("title")
				actor, _ := record.Get("actor")
				actorRole, _ := record.Get("role")
				roles = append(roles, &Role{
					Title: title.(string),
					Actor: actor.(string),
					Role:  actorRole.(string),
				})
			}
			return roles, nil
			// Option 2: we collect all records from result and iterate and map outside of the transaction
			// return result.Collect(ctx)
		})
	if err != nil {
		mr.logger.Println("Error querying search:", err)
		return nil, err
	}
	return roleResults.(Roles), nil
}

// Create movie
func (mr *MovieRepo) WriteMovie(movie *Movie) error {
	ctx := context.Background()
	session := mr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	savedMovie, err := session.ExecuteWrite(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				"CREATE (m:Movie) SET m.title = $title, m.released = $released, m.tagline = $tagline RETURN m.title + ', from node ' + id(m)",
				map[string]any{"title": movie.Title, "released": movie.Released, "tagline": movie.Tagline})
			if err != nil {
				return nil, err
			}

			if result.Next(ctx) {
				return result.Record().Values[0], nil
			}

			return nil, result.Err()
		})
	if err != nil {
		mr.logger.Println("Error inserting Movie:", err)
		return err
	}
	mr.logger.Println(savedMovie.(string))
	return nil
}

// Actor and producer
func (mr *MovieRepo) GetPersonWhoActedAndProducedMovie() (Roles, error) {
	ctx := context.Background()
	session := mr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	roleResults, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				`MATCH (p:Person)-[a:ACTED_IN]->(m:Movie)<-[d:DIRECTED]-(p)
				RETURN p.name as actor, m.title as title, a.roles[0] as role`,
				map[string]any{})
			if err != nil {
				return nil, err
			}

			var roles Roles
			for result.Next(ctx) {
				record := result.Record()
				title, _ := record.Get("title")
				actor, _ := record.Get("actor")
				actorRole, _ := record.Get("role")
				roles = append(roles, &Role{
					Title: title.(string),
					Actor: actor.(string),
					Role:  actorRole.(string),
				})
			}
			return roles, nil
		})
	if err != nil {
		mr.logger.Println("Error querying search:", err)
		return nil, err
	}
	return roleResults.(Roles), nil
}

// Keanu Reeves movies
func (mr *MovieRepo) GetKeanuMovies(limit int) (Movies, error) {
	ctx := context.Background()
	session := mr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	moviesResult, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				`MATCH (p:Person {name: "Keanu Reeves"}) -[a:ACTED_IN]-> (m:Movie)
				RETURN m.title as title, m.tagline as tagline, m.released as released
				ORDER BY m.released DESC
				LIMIT $limit`,
				map[string]any{"limit": limit})
			if err != nil {
				return nil, err
			}

			var movies Movies
			for result.Next(ctx) {
				record := result.Record()
				released, ok := record.Get("released")
				if !ok || released == nil {
					released = 0
				}
				title, _ := record.Get("title")
				tagline, ok := record.Get("tagline")
				if !ok || tagline == nil {
					tagline = ""
				}
				movies = append(movies, &Movie{
					Released: released.(int64),
					Title:    title.(string),
					Tagline:  tagline.(string),
				})
			}
			return movies, nil
		})
	if err != nil {
		mr.logger.Println("Error querying search:", err)
		return nil, err
	}
	return moviesResult.(Movies), nil
}

// Shorthest path query
// match path = shortestPath((matrix:Movie {title: "The Matrix"})-[*]-(harry:Movie {title: "When Harry Met Sally"}))
// return path

// Actors with most movies
func (mr *MovieRepo) GetActorsWithMostMovies(limit int) (ActorsMostMovies, error) {
	ctx := context.Background()
	session := mr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	actorsResult, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				`MATCH (actor:Person)-[:ACTED_IN]-(movie:Movie) 
				RETURN 
					actor.name as actor,
					COLLECT(DISTINCT movie.title) as movies, 
					COUNT(DISTINCT movie) as cnt 
				ORDER BY cnt DESC 
				LIMIT $limit`,
				map[string]any{"limit": limit})
			if err != nil {
				return nil, err
			}

			var actors ActorsMostMovies
			for result.Next(ctx) {
				record := result.Record()
				actor, _ := record.Get("actor")
				cnt, _ := record.Get("cnt")
				actors = append(actors, &ActorMostMovies{
					Actor: actor.(string),
					Count: cnt.(int64),
				})
			}
			return actors, nil
		})
	if err != nil {
		mr.logger.Println("Error querying search:", err)
		return nil, err
	}
	return actorsResult.(ActorsMostMovies), nil
}

// func (mr *MovieRepo) convertDataToFloat64(data any) (float64) {
// 	switch i := data.(type) {
//     case float64:
//             return i
//     case float32:
//             return float64(i)
//     case int64:
//             return float64(i)
//     default:
//             return 0.0
//     }
// }

func (mr *MovieRepo) convertDataToPersonSlice(data any) []Person {
	var people []Person
	if data == nil {
		return people
	}
	list := data.([]interface{})
	if len(list) == 0 {
		return people
	}
	for _, v := range list {
		if v == nil {
			continue
		}
		people = append(people, Person{
			Name: v.(string),
		})
	}
	return people
}
