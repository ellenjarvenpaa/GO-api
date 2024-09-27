package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Animal struct {
	ID         primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	AnimalName string             `json:"animal_name" bson:"animal_name"`
	Species    string             `json:"species" bson:"species"`
	Birthdate  time.Time          `json:"birthdate" bson:"birthdate"`
	Location   struct {
		Type        string     `json:"type" bson:"type"`
		Coordinates [2]float64 `json:"coordinates" bson:"coordinates"`
	} `json:"location" bson:"location"`
	Owner   string `json:"owner" bson:"owner"`
	Version int    `json:"__v" bson:"__v"`
}

var collection *mongo.Collection

func main() {
	fmt.Println("Hello World")

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}

	MONGODB_URI := os.Getenv("MONGODB_URI")
	clientOptions := options.Client().ApplyURI(MONGODB_URI)
	client, err := mongo.Connect(context.Background(), clientOptions)

	if err != nil {
		log.Fatal(err)
	}

	defer client.Disconnect(context.Background())

	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MONGODB ATLAS")

	collection = client.Database("palvelinohjelmointi").Collection("animals")

	app := fiber.New()

	app.Get("/api/animals", getAnimals)
	app.Post("/api/animals", postAnimals)
	app.Put("/api/animals/:id", putAnimals)
	app.Delete("/api/animals/:id", deleteAnimals)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))

}

func getAnimals(c *fiber.Ctx) error {
	var animals []Animal

	cursor, err := collection.Find(context.Background(), bson.M{})

	if err != nil {
		return err
	}

	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var animal Animal
		if err := cursor.Decode(&animal); err != nil {
			return err
		}
		animals = append(animals, animal)
	}

	return c.JSON(animals)
}

func postAnimals(c *fiber.Ctx) error {
	animal := new(Animal)

	if err := c.BodyParser(animal); err != nil {
		return err
	}

	if animal.AnimalName == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Animal name cannot be empty"})
	}

	insertResult, err := collection.InsertOne(context.Background(), animal)
	if err != nil {
		return err
	}

	animal.ID = insertResult.InsertedID.(primitive.ObjectID)

	return c.Status(201).JSON(fiber.Map{"message": "Animal added successfully"})
}

func putAnimals(c *fiber.Ctx) error {
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid animal ID"})
	}

	var animalUpdate Animal
	if err := c.BodyParser(&animalUpdate); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Failed to parse request body"})
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{
		"$set": bson.M{
			"animal_name": animalUpdate.AnimalName,
			"species":     animalUpdate.Species,
			"birthdate":   animalUpdate.Birthdate,
			"location": bson.M{
				"type":        animalUpdate.Location.Type,
				"coordinates": animalUpdate.Location.Coordinates,
			},
			"owner": animalUpdate.Owner,
		},
	}

	result, err := collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update animal"})
	}

	if result.MatchedCount == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "Animal not found"})
	}

	return c.JSON(fiber.Map{
		"message":        "Animal updated successfully",
		"modified_count": result.ModifiedCount,
	})
}

func deleteAnimals(c *fiber.Ctx) error {
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid ID"})
	}

	filter := bson.M{"_id": objectID}
	_, err = collection.DeleteOne(context.Background(), filter)

	if err != nil {
		return err
	}

	return c.Status(200).JSON(fiber.Map{"message": "Animal deleted successfully"})

}
