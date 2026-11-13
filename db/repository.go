package db

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// URL represents the data model stored in MongoDB
type URL struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	OriginalURL  string        `bson:"original_url" json:"original_url"`
	ShortURL     string        `bson:"short_url" json:"short_url"`
	CreationDate time.Time     `bson:"creation_date" json:"creation_date"`
}

// EnsureIndexes creates unique indexes on short_url and original_url
// to prevent hash collisions and concurrent duplicate URL submissions.
func EnsureIndexes(ctx context.Context, collection *mongo.Collection) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "short_url", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "original_url", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// InsertURL saves a new short URL document into the database
func InsertURL(ctx context.Context, collection *mongo.Collection, originalURL, shortURL string) error {
	newURL := URL{
		OriginalURL:  originalURL,
		ShortURL:     shortURL,
		CreationDate: time.Now(),
	}

	_, err := collection.InsertOne(ctx, newURL)
	return err
}

// GetURLByShortCode searches for a URL document by its short code ID
func GetURLByShortCode(ctx context.Context, collection *mongo.Collection, shortCode string) (URL, error) {
	var result URL
	filter := bson.M{"short_url": shortCode}

	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return URL{}, errors.New("URL not found!")
		}
		return URL{}, err
	}

	return result, nil
}

// GetURLByOriginal searches for an existing URL document by its original URL
func GetURLByOriginal(ctx context.Context, collection *mongo.Collection, originalURL string) (URL, error) {
	var result URL
	filter := bson.M{"original_url": originalURL}

	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return URL{}, mongo.ErrNoDocuments
		}
		return URL{}, err
	}

	return result, nil
}
