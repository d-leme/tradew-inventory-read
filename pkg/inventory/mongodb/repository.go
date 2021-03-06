package mongodb

import (
	"context"
	"time"

	"github.com/d-leme/tradew-inventory-read/pkg/inventory"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

type repositoryMongoDB struct {
	collection *mongo.Collection
}

// NewRepository ...
func NewRepository(client *mongo.Client, database string) inventory.Repository {
	repository := &repositoryMongoDB{client.Database(database).Collection("inventory")}
	repository.createIndex()

	return repository
}

// Get ...
func (repository *repositoryMongoDB) Get(ctx context.Context, req *inventory.GetItemsRequest) (*inventory.GetItemsResponse, error) {

	if req.PageSize < 1 {
		req.PageSize = 10
	}

	result := new(inventory.GetItemsResponse)
	result.Items = []*inventory.Item{}

	filter := bson.M{}
	if req.Token != nil {
		filter["_id"] = bson.M{"$gt": req.Token}
	}

	cursor, err := repository.collection.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.M{"_id": 1}).SetLimit(req.PageSize),
	)

	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	err = cursor.All(ctx, &result.Items)
	if err != nil {
		return nil, err
	}

	if lastItem := result.Items[len(result.Items)-1]; lastItem != nil {
		result.Token = lastItem.ID
	}

	return result, nil
}

// GetByID ...
func (repository *repositoryMongoDB) GetByID(ctx context.Context, userID string, id string) (*inventory.Item, error) {
	var r *inventory.Item

	err := repository.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&r)

	if err != nil {
		return nil, err
	}

	return r, nil
}

// GetByIDs ...
func (repository *repositoryMongoDB) GetByIDs(ctx context.Context, ids []string) ([]*inventory.Item, error) {

	var items []*inventory.Item

	cursor, err := repository.collection.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})

	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// GetUserItems ...
func (repository *repositoryMongoDB) GetUserItems(ctx context.Context, userID string, req *inventory.GetItemsRequest) (*inventory.GetItemsResponse, error) {
	if req.PageSize < 1 {
		req.PageSize = 10
	}

	result := new(inventory.GetItemsResponse)
	result.Items = []*inventory.Item{}

	filter := bson.M{"owner_id": userID}

	if req.Token != nil {
		filter["_id"] = bson.M{"$gt": req.Token}
	}

	cursor, err := repository.collection.Find(
		ctx,
		filter,
		options.Find().SetSort(bson.M{"_id": 1}).SetLimit(req.PageSize),
	)

	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	err = cursor.All(ctx, &result.Items)
	if err != nil {
		return nil, err
	}

	if lastItem := result.Items[len(result.Items)-1]; lastItem != nil {
		result.Token = lastItem.ID
	}

	return result, nil
}

// InsertBulk ...
func (repository *repositoryMongoDB) InsertBulk(ctx context.Context, items []*inventory.Item) error {
	documents := make([]interface{}, len(items))

	for i, item := range items {
		documents[i] = item
	}

	if _, err := repository.collection.InsertMany(ctx, documents); err != nil {
		return err
	}

	return nil
}

// UpdateBulk ...
func (repository *repositoryMongoDB) UpdateBulk(ctx context.Context, items []*inventory.Item) error {

	models := make([]mongo.WriteModel, len(items))

	for i, item := range items {
		model := mongo.NewUpdateOneModel()
		model.SetFilter(bson.M{"_id": item.ID})
		model.SetUpdate(bson.M{"$set": item})
		model.SetUpsert(true)
		models[i] = model
	}

	_, err := repository.collection.BulkWrite(ctx, models)

	return err
}

func (repository *repositoryMongoDB) createIndex() {
	ctx, close := context.WithTimeout(context.Background(), 10*time.Second)
	defer close()

	filterName := mongo.IndexModel{
		Keys: bsonx.Doc{
			{Key: "name", Value: bsonx.Int32(-1)},
		},
		Options: options.Index().SetUnique(false),
	}

	repository.collection.Indexes().CreateOne(ctx, filterName)
}
