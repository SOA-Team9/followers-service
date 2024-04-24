package repo

import (
	"context"
	"errors"
	"log"
	"os"

	"followers-service.xws.com/model"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type FollowRepo struct {
	driver neo4j.DriverWithContext
	logger *log.Logger
}

func NewFollowsStore(logger *log.Logger) (*FollowRepo, error) {

	uri := os.Getenv("NEO4J_DB")
	user := os.Getenv("NEO4J_USERNAME")
	password := os.Getenv("NEO4J_PASS")
	auth := neo4j.BasicAuth(user, password, "")

	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		return nil, err
	}
	return &FollowRepo{driver: driver, logger: logger}, nil
}

func (fr *FollowRepo) CheckConnection() {
	ctx := context.Background()
	err := fr.driver.VerifyConnectivity(ctx)
	if err != nil {
		fr.logger.Panic(err)
		return
	}
	fr.logger.Printf(`Neo4J server address: %s`, fr.driver.Target().Host)
}

func (fr *FollowRepo) CloseDriverConnection(ctx context.Context) {
	fr.driver.Close(ctx)
}

func (fr *FollowRepo) FollowUser(followerID int, followedID int) (model.Follow, error) {
	ctx := context.Background()
	session := fr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	// Check if the relationship already exists
	exists, err := fr.checkFollowRelationship(ctx, session, followerID, followedID)
	if err != nil {
		return model.Follow{}, err
	}
	if exists {
		return model.Follow{}, errors.New("relationship already exists")
	}

	// Create the relationship
	_, err = session.ExecuteWrite(ctx,
		func(transaction neo4j.ManagedTransaction) (interface{}, error) {
			_, err := transaction.Run(ctx,
				`MATCH (follower:User {Id: $followerID}), (followed:User {Id: $followedID})
                CREATE (follower)-[:FOLLOWS]->(followed)`,
				map[string]interface{}{"followerID": followerID, "followedID": followedID})
			if err != nil {
				return nil, err
			}
			return nil, nil
		})
	if err != nil {
		fr.logger.Println("Error creating follow:", err)
		return model.Follow{}, err
	}

	follow := model.Follow{
		FollowerID: followerID,
		FollowedID: followedID,
	}

	return follow, nil
}

func (fr *FollowRepo) checkFollowRelationship(ctx context.Context, session neo4j.SessionWithContext, followerID int, followedID int) (bool, error) {
	result, err := session.ExecuteRead(ctx, func(transaction neo4j.ManagedTransaction) (interface{}, error) {
		query := `
			MATCH (follower:User {Id: $followerID})-[:FOLLOWS]->(followed:User {Id: $followedID})
			RETURN COUNT(*) > 0
		`
		params := map[string]interface{}{"followerID": followerID, "followedID": followedID}
		cursor, err := transaction.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		if cursor.Next(ctx) {
			record := cursor.Record()
			exists, ok := record.Values[0].(bool)
			if !ok {
				return nil, errors.New("invalid result for follow relationship check")
			}
			return exists, nil
		}

		return false, nil
	})
	if err != nil {
		return false, err
	}

	return result.(bool), nil
}

func (fr *FollowRepo) CheckFollow(followerID int, followedID int) (bool, error) {
	ctx := context.Background()
	session := fr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (interface{}, error) {
			result, err := transaction.Run(ctx,
				`MATCH (follower:User {Id: $followerID})-[r:FOLLOWS]->(followed:User {Id: $followedID})
			RETURN r`,
				map[string]interface{}{"followerID": followerID, "followedID": followedID})
			if err != nil {
				return nil, err
			}

			if result.Next(ctx) {
				return result.Record().Values[0], nil
			}

			return nil, result.Err()
		})
	if err != nil {
		fr.logger.Println("Error checking follow:", err)
		return false, err
	}

	if result != nil {
		return true, nil
	}

	return false, nil
}

func (ur *FollowRepo) AddUser(user *model.User) error {
	ctx := context.Background()
	session := ur.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	savedUser, err := session.ExecuteWrite(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				"CREATE (u:User) SET u.Id = $id, u.Username = $username RETURN u.Username + ', from node ' + id(u)",
				map[string]any{"id": user.Id, "username": user.Username})
			if err != nil {
				return nil, err
			}

			if result.Next(ctx) {
				return result.Record().Values[0], nil
			}

			return nil, result.Err()
		})
	if err != nil {
		ur.logger.Println("Error inserting User:", err)
		return err
	}
	if savedUser != nil {
		ur.logger.Println(savedUser.(string))
	}
	return nil
}

func (fr *FollowRepo) GetUserFollowing(userId int) ([]model.Follow, error) {
	ctx := context.Background()
	session := fr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	following, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (interface{}, error) {
			result, err := transaction.Run(ctx,
				`MATCH (u:User {Id: $userId})-[:FOLLOWS]->(f:User)
                RETURN u.Id, f.Id`,
				map[string]interface{}{"userId": userId})
			if err != nil {
				return nil, err
			}

			var follows []model.Follow
			for result.Next(ctx) {
				record := result.Record()
				followerID := int(record.Values[0].(int64))
				followedID := int(record.Values[1].(int64))
				follow := model.Follow{
					FollowerID: followerID,
					FollowedID: followedID,
				}
				follows = append(follows, follow)
			}

			return follows, result.Err()
		})
	if err != nil {
		fr.logger.Println("Error getting following:", err)
		return nil, err
	}

	if following != nil {
		fr.logger.Println(following.([]model.Follow))
		return following.([]model.Follow), nil
	}
	return nil, nil
}

func (fr *FollowRepo) GetUserFollowingIds(userId int) ([]int64, error) {
	ctx := context.Background()
	session := fr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	following, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (interface{}, error) {
			result, err := transaction.Run(ctx,
				`MATCH (u:User {Id: $userId})-[:FOLLOWS]->(f:User)
                RETURN collect(f.Id) AS followedIds`,
				map[string]interface{}{"userId": userId})
			if err != nil {
				return nil, err
			}

			var followedIds []int64
			for result.Next(ctx) {
				record := result.Record()
				followedIDs := record.Values[0].([]interface{})
				for _, id := range followedIDs {
					followedId := id.(int64)
					followedIds = append(followedIds, followedId)
				}
			}

			return map[string][]int64{"followedIds": followedIds}, result.Err()
		})
	if err != nil {
		fr.logger.Println("Error getting following:", err)
		return nil, err
	}

	if following != nil {
		followingMap := following.(map[string][]int64)
		fr.logger.Println("Followed IDs:", followingMap["followedIds"])
		return followingMap["followedIds"], nil
	}
	return nil, nil
}

func (fr *FollowRepo) GetUserFollowers(userId int) ([]model.Follow, error) {
	ctx := context.Background()
	session := fr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	followers, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (interface{}, error) {
			result, err := transaction.Run(ctx,
				`MATCH (u:User {Id: $userId})<-[:FOLLOWS]-(f:User)
                RETURN f.Id, u.Id`,
				map[string]interface{}{"userId": userId})
			if err != nil {
				return nil, err
			}

			var followerList []model.Follow
			for result.Next(ctx) {
				record := result.Record()
				followerID := int(record.Values[0].(int64))
				followedID := int(record.Values[1].(int64))
				follower := model.Follow{
					FollowerID: followerID,
					FollowedID: followedID,
				}
				followerList = append(followerList, follower)
			}

			return followerList, result.Err()
		})
	if err != nil {
		fr.logger.Println("Error getting followers:", err)
		return nil, err
	}

	if followers != nil {
		fr.logger.Println(followers.([]model.Follow))
		return followers.([]model.Follow), nil
	}
	return nil, nil
}

func (fr *FollowRepo) GetFollowRecommendations(userID int) ([]int64, error) {
	ctx := context.Background()
	session := fr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (interface{}, error) {
			result, err := transaction.Run(ctx,
				`MATCH (u:User {Id: $userID})-[:FOLLOWS]->(:User)-[:FOLLOWS]->(recommendation:User)
				WHERE NOT (u)-[:FOLLOWS]->(recommendation) AND u <> recommendation
				RETURN DISTINCT recommendation.Id
				`,
				map[string]interface{}{"userID": userID})

			if err != nil {
				return nil, err
			}

			var recommendations []int64
			for result.Next(ctx) {
				record := result.Record()
				recommendationID, found := record.Get("recommendation.Id")
				if !found {
					continue
				}
				recommendations = append(recommendations, recommendationID.(int64))
			}

			return recommendations, nil
		})

	if err != nil {
		fr.logger.Println("Error getting follow recommendations:", err)
		return nil, err
	}

	if recommendations, ok := result.([]int64); ok {
		if len(recommendations) < 10 {
			additionalRecommendations, err := fr.getAdditionalRecommendations(ctx, session, userID, len(recommendations), 10)
			if err != nil {
				fr.logger.Println("Error getting additional follow recommendations:", err)
				return nil, err
			}
			recommendationsTemp := recommendations
			for _, value := range additionalRecommendations {
				exists := false
				for _, value2 := range recommendationsTemp {
					if value == value2 {
						exists = true
						break
					}
				}
				if !exists {
					recommendations = append(recommendations, value)
				}
			}
			//recommendations = append(recommendations, additionalRecommendations...)
		}
		return recommendations, nil
	}

	return nil, nil
}

func (fr *FollowRepo) getAdditionalRecommendations(ctx context.Context, session neo4j.SessionWithContext, userID, currentCount, targetCount int) ([]int64, error) {
	result, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (interface{}, error) {
			result, err := transaction.Run(ctx,
				`MATCH (u:User {Id: $userID})
				OPTIONAL MATCH (recommendation:User)
				WHERE recommendation.Id <> $userID AND NOT (u)-[:FOLLOWS]->(recommendation)
				RETURN recommendation.Id
				`,
				map[string]interface{}{"userID": userID, "limit": targetCount - currentCount})

			if err != nil {
				return nil, err
			}

			var additionalRecommendations []int64
			for result.Next(ctx) {
				record := result.Record()
				recommendationID, found := record.Get("recommendation.Id")
				if !found {
					continue
				}
				additionalRecommendations = append(additionalRecommendations, recommendationID.(int64))
			}

			return additionalRecommendations, nil
		})

	if err != nil {
		return nil, err
	}

	if recommendations, ok := result.([]int64); ok {
		return recommendations, nil
	}

	return nil, nil
}

func (fr *FollowRepo) UnfollowUser(follow model.Follow) error {
	ctx := context.Background()
	session := fr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx,
		func(transaction neo4j.ManagedTransaction) (interface{}, error) {
			_, err := transaction.Run(ctx,
				`MATCH (follower:User {Id: $followerID})-[r:FOLLOWS]->(followed:User {Id: $followedID})
				DELETE r`,
				map[string]interface{}{"followerID": follow.FollowerID, "followedID": follow.FollowedID})
			if err != nil {
				return nil, err
			}
			return nil, nil
		})

	if err != nil {
		fr.logger.Println("Error unfollowing user:", err)
		return err
	}

	return nil
}
