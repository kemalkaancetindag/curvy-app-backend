package main

import (
	"context"
	"fmt"
	"log"
	math "math"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func recommend(client firestore.Client, ctx context.Context, userID string, unLikedUsers []string) []map[string]interface{} {

	userIterator := client.Collection("users").Where("userID", "==", userID).Documents(ctx)
	responseSlice := []map[string]interface{}{}

	for {
		doc, err := userIterator.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}

		var user = doc.Data()

		if err == nil {
			iter := client.Collection("users").Where("userID", "!=", userID).Documents(ctx)

			for {
				doc, err := iter.Next()

				if err == iterator.Done {
					break
				}
				if err != nil {
					log.Fatalf("Failed to iterate: %v", err)
				}

				var user2 = doc.Data()
				contains := slices.Contains(unLikedUsers, user2["userID"].(string))
				if contains {
					continue
				}

				ageBool := agePreferenceDecider(user, user2)
				sexBool := sexPreferenceDecider(user, user2)
				distanceBool := distancePreferenceDecider(user, user2)

				if ageBool && sexBool && distanceBool {
					fmt.Println(user2)
					responseSlice = append(responseSlice, user2)
				}

			}
		}

	}

	if len(responseSlice) < 10 {
		return responseSlice
	}
	return responseSlice[:10]

}

func agePreferenceDecider(user, user2 map[string]interface{}) bool {

	birthYear, err := strconv.Atoi(strings.Split(user2["birthdate"].(string), "/")[2])

	if err == nil {
		year, _, _ := time.Now().Date()
		currentUserAge := year - birthYear
		settingsInterface := user["settings"].(map[string]interface{})
		agePreferenceInterface := settingsInterface["age_preference"].(map[string]interface{})
		minAge := agePreferenceInterface["min_age"].(int64)
		maxAge := agePreferenceInterface["max_age"].(int64)

		if currentUserAge >= int(minAge) && currentUserAge <= int(maxAge) {
			return true
		}

		return false

	} else {

	}

	return false

}

func sexPreferenceDecider(user, user2 map[string]interface{}) bool {
	//sex-showme
	//0 Kadın
	// 1 Erkek
	// 2 Hepsi
	user2Sex := user2["sex"].(int64)
	userShowMe := user["show_me"].(int64)

	if userShowMe == 2 {
		return true
	} else if userShowMe == user2Sex {
		return true
	}

	return false

}

func distancePreferenceDecider(user, user2 map[string]interface{}) bool {
	userLocationObject := user["location"]
	user2LocationObject := user2["location"]

	if userLocationObject != nil && user2LocationObject != nil {
		userLocationInterface := userLocationObject.(map[string]interface{})
		user2LocationInterface := user2LocationObject.(map[string]interface{})
		userLatitude := userLocationInterface["latitude"].(float64)
		userLongitude := userLocationInterface["longitude"].(float64)
		user2Latitude := user2LocationInterface["latitude"].(float64)
		user2Longitude := user2LocationInterface["longitude"].(float64)

		distance := calculteDistance(userLongitude, userLatitude, user2Longitude, user2Latitude)

		userSettingsInterface := user["settings"].(map[string]interface{})
		userDistancePreferenceInterface := userSettingsInterface["distance_preference"].(map[string]interface{})
		userDistancePreference := userDistancePreferenceInterface["distance"].(float64)

		if distance <= userDistancePreference {
			return true
		}

		return false

	}
	return false

}

func calculteDistance(lon1 float64, lat1 float64, lon2 float64, lat2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	latTimesPi1 := (lat1) * math.Pi / 180.0
	latTimesPi2 := (lat2) * math.Pi / 180.0

	a := (math.Pow(math.Sin(dLat/2), 2) +
		math.Pow(math.Sin(dLon/2), 2)*
			math.Cos(latTimesPi1)*math.Cos(latTimesPi2))
	rad := 6371
	c := 2 * math.Asin(math.Sqrt(a))

	return float64(rad) * c
}

func main() {

	ctx := context.Background()
	sa := option.WithCredentialsFile("./curvy-cred.json")
	app, err := firebase.NewApp(ctx, nil, sa)

	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()

	router := gin.Default()

	router.POST("/recommendations", func(c *gin.Context) {
		//jsonData, err := ioutil.ReadAll(c.Request.Body)
		if err == nil {
			var requestBody RecommendationsRequestBody
			c.BindJSON(&requestBody)
			fmt.Println(requestBody.UserID)
			fmt.Println(requestBody.UnLikedUsers)
			results := recommend(*client, ctx, requestBody.UserID, requestBody.UnLikedUsers)

			c.JSON(200, results)
		}

	})

	router.Run("localhost:8080")

}

type RecommendationsRequestBody struct {
	UserID       string   `json:"userID"`
	UnLikedUsers []string `json:"un_liked_users"`
}
