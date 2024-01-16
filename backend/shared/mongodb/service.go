// mongodb/mongodb.go
package mongodb

import (
	"context"
	"errors"
	"log"
	"shared/models"
	"shared/utils"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoService defines the interface for interacting with MongoDB.
type MongoService interface {
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	InsertUser(ctx context.Context, user models.User) (*mongo.InsertOneResult, error)
	DeleteUserByEmail(ctx context.Context, email string) (*mongo.DeleteResult, error)
	UpdateUserDetails(ctx context.Context, userId primitive.ObjectID, updatedUserDetails models.User) error
	CreateEvent(ctx context.Context, event models.Event) (*mongo.InsertOneResult, error)
	DeleteEvent(ctx *gin.Context, eventID primitive.ObjectID) (*mongo.DeleteResult, error)
	GetEvent(ctx *gin.Context, eventID primitive.ObjectID) (*models.Event, error)
	UpdateEventMetadata(ctx *gin.Context, eventID primitive.ObjectID, metadata models.EventMetadata) (*mongo.UpdateResult, error)
	ListEventsMetadata(ctx context.Context, filter bson.M) ([]models.Event, error)
	CreateSource(ctx context.Context, source models.SelectorSource) (*mongo.InsertOneResult, error)
	UpdateSource(ctx context.Context, source models.SelectorSource, sourceID primitive.ObjectID) (*mongo.UpdateResult, error)
	GetSourceByName(ctx context.Context, name string) (*models.SelectorSource, error)
	GetForm(ctx context.Context, formID primitive.ObjectID) (*models.FormStructure, error)
	ListForms(ctx context.Context, filter bson.M) ([]models.FormStructure, error)
	CreateForm(ctx context.Context, form models.FormStructure) (*mongo.InsertOneResult, error)
	UpdateForm(ctx context.Context, form models.FormStructure, formID primitive.ObjectID) (*mongo.UpdateResult, error)
	DeleteForm(ctx context.Context, formID primitive.ObjectID) (*mongo.DeleteResult, error)
	CreatePipeline(ctx context.Context, pipeline models.PipelineConfiguration) (*mongo.InsertOneResult, error)
	UpdatePipeline(ctx context.Context, pipeline models.PipelineConfiguration, pipelineID primitive.ObjectID) (*mongo.UpdateResult, error)
	GetPipeline(ctx context.Context, pipelineID primitive.ObjectID) (*models.PipelineConfiguration, error)
	ListPipelines(ctx context.Context, filter bson.M) ([]models.PipelineConfiguration, error)
	DeletePipeline(ctx context.Context, pipelineID primitive.ObjectID) (*mongo.DeleteResult, error)
}

// Service implements MongoService with a mongo.Client.
type Service struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewService creates a new Service.
func NewService() (*Service, func(), error) {
	// TODO: We might want to initialize the connection lazily
	client, err := getMongoClient()
	if err != nil {
		return nil, nil, err
	}

	// Cleanup function
	cleanup := func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting from MongoDB: %s", err)
		}
	}

	database := client.Database(MongoDBName)
	return &Service{Client: client, Database: database}, cleanup, nil
}

// FindUserByEmail finds a user by their email.
func (s *Service) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.Database.Collection("users").FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// InsertUser inserts a new user into the database.
func (s *Service) InsertUser(ctx context.Context, user models.User) (*mongo.InsertOneResult, error) {
	// First check if email is already registered
	_, err := s.FindUserByEmail(ctx, user.Email)
	if err == nil {
		return nil, ErrUserAlreadyExists
	}
	return s.Database.Collection("users").InsertOne(ctx, user)
}

// DeleteUserByEmail deletes a user by their email.
func (s *Service) DeleteUserByEmail(ctx context.Context, email string) (*mongo.DeleteResult, error) {
	return s.Database.Collection("users").DeleteOne(ctx, bson.M{"email": email})
}

// Create a new event
func (s *Service) CreateEvent(ctx context.Context, event models.Event) (*mongo.InsertOneResult, error) {
	// First make sure the name exists
	if event.Metadata.Name == "" {
		return nil, ErrEventNameRequired
	}

	// Ensure the event is not visible when first created
	event.Metadata.Visibility = false
	return s.Database.Collection("events").InsertOne(ctx, event)
}

// Update an event by its ID but only if the user is an organizer
func (s *Service) UpdateEventMetadata(ctx *gin.Context, eventID primitive.ObjectID, metadata models.EventMetadata) (*mongo.UpdateResult, error) {
	authenticatedUser, ok := utils.GetUserFromContext(ctx, true)
	if !ok {
		return nil, ErrUserNotAuthenticated
	}

	// Lookup the event
	var event models.Event
	err := s.Database.Collection("events").FindOne(ctx, bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		return nil, err
	}

	if !CanUserModifyEvent(ctx, s, authenticatedUser, event.ID, &event) {
		return nil, ErrUserNotAuthorized
	}

	// Update the event metadata in the database
	update := bson.M{"$set": bson.M{"metadata": metadata}}
	return s.Database.Collection("events").UpdateOne(ctx, bson.M{"_id": eventID}, update)
}

type EventMetadataWithID struct {
	ID       primitive.ObjectID `json:"id"`
	Metadata models.EventMetadata
}

// ListEventsMetadata retrieves events based on a filter
func (s *Service) ListEventsMetadata(ctx context.Context, filter bson.M) ([]models.Event, error) {
	var events []models.Event

	cursor, err := s.Database.Collection("events").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var event models.Event
		if err := cursor.Decode(&event); err != nil {
			return nil, err
		}

		// We re-create the event here because we don't want to return the organizer IDs or hidden fields
		events = append(events, models.Event{
			ID:       event.ID,
			Metadata: event.Metadata,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	// If events is null then return an empty slice instead
	if events == nil {
		return []models.Event{}, nil
	}

	return events, nil
}

// DeleteEvent deletes an event by its ID
func (s *Service) DeleteEvent(ctx *gin.Context, eventID primitive.ObjectID) (*mongo.DeleteResult, error) {
	authenticatedUser, ok := utils.GetUserFromContext(ctx, true)
	if !ok {
		return nil, ErrUserNotAuthenticated
	}

	// Lookup the event
	var event models.Event
	err := s.Database.Collection("events").FindOne(ctx, bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		return nil, err
	}

	// Ensure the user is an organizer
	if !CanUserModifyEvent(ctx, s, authenticatedUser, event.ID, &event) {
		return nil, ErrUserNotAuthorized
	}

	// Delete the event
	return s.Database.Collection("events").DeleteOne(ctx, bson.M{"_id": eventID})
}

// GetEvent retrieves an event by its ID
// Returns metadata if the user is not an organizer (through ListEventsMetadata)
// Returns the full event if the user is an organizer
func (s *Service) GetEvent(ctx *gin.Context, eventID primitive.ObjectID) (*models.Event, error) {
	authenticatedUser, isAuthenticated := utils.GetUserFromContext(ctx, false)
	if authenticatedUser == nil || authenticatedUser.ID == primitive.NilObjectID {
		isAuthenticated = false
	}

	// Lookup the event
	var event models.Event
	err := s.Database.Collection("events").FindOne(ctx, bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		return nil, err
	}

	// If the user is not an organizer then return the metadata
	if !isAuthenticated || !CanUserModifyEvent(ctx, s, authenticatedUser, event.ID, &event) {
		return &models.Event{
			ID:       event.ID,
			Metadata: event.Metadata,
		}, nil
	}

	return &event, nil
}

// GetEventForms returns a list of all of the events forms
func (s *Service) ListForms(ctx context.Context, filter bson.M) ([]models.FormStructure, error) {
	var forms []models.FormStructure
	cursor, err := s.Database.Collection("forms").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var form models.FormStructure
		if err := cursor.Decode(&form); err != nil {
			return nil, err
		}
		forms = append(forms, form)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	if forms == nil {
		return []models.FormStructure{}, nil
	}

	return forms, nil
}

// UpdateUserDetails updates a user given the user's ObjectId
func (s *Service) UpdateUserDetails(ctx context.Context, userId primitive.ObjectID, updatedUserDetails models.User) error {
	// Update the user details in the database
	update := bson.M{"$set": bson.M{
		"firstName":   updatedUserDetails.FirstName,
		"lastName":    updatedUserDetails.LastName,
		"schoolEmail": updatedUserDetails.SchoolEmail,
		"birthday":    updatedUserDetails.Birthday,
	}}

	filter := bson.M{"_id": userId}
	result, err := s.Database.Collection("users").UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("no user found with that ID")
	}

	return nil
}

// GetSourceByName retrieves a SelectorSource by its name
func (s *Service) GetSourceByName(ctx context.Context, name string) (*models.SelectorSource, error) {
	var source models.SelectorSource
	err := s.Database.Collection("sources").FindOne(ctx, bson.M{"sourceName": name}).Decode(&source)
	if err != nil {
		return nil, err
	}
	return &source, nil
}

func (s *Service) CreateSource(ctx context.Context, source models.SelectorSource) (*mongo.InsertOneResult, error) {
	_, err := s.GetSourceByName(ctx, source.SourceName)
	if err == nil {
		return nil, errors.New("source already exists")
	}
	return s.Database.Collection("sources").InsertOne(ctx, source)
}

func (s *Service) UpdateSource(ctx context.Context, source models.SelectorSource, sourceID primitive.ObjectID) (*mongo.UpdateResult, error) {
	update := bson.M{"$set": bson.M{
		"sourceName":  source.SourceName,
		"lastUpdated": source.LastUpdated,
		"options":     source.Options,
	}}

	filter := bson.M{"_id": sourceID}
	return s.Database.Collection("sources").UpdateOne(ctx, filter, update)
}

// GetForm retrieves a form by its ID
func (s *Service) GetForm(ctx context.Context, formID primitive.ObjectID) (*models.FormStructure, error) {
	var form models.FormStructure
	err := s.Database.Collection("forms").FindOne(ctx, bson.M{"_id": formID}).Decode(&form)
	if err != nil {
		return nil, err
	}
	return &form, nil
}

// CreateForm creates a new form
func (s *Service) CreateForm(ctx context.Context, form models.FormStructure) (*mongo.InsertOneResult, error) {
	form.CreatedAt = time.Now()
	form.UpdatedAt = time.Now()
	form.IsDeleted = false
	form.Status = "draft"
	return s.Database.Collection("forms").InsertOne(ctx, form)
}

// UpdateForm updates a form by its ID
func (s *Service) UpdateForm(ctx context.Context, form models.FormStructure, formID primitive.ObjectID) (*mongo.UpdateResult, error) {
	form.UpdatedAt = time.Now()
	update := bson.M{"$set": form}
	filter := bson.M{"_id": formID}
	return s.Database.Collection("forms").UpdateOne(ctx, filter, update)
}

// DeleteForm deletes a form by its ID
func (s *Service) DeleteForm(ctx context.Context, formID primitive.ObjectID) (*mongo.DeleteResult, error) {
	filter := bson.M{"_id": formID}
	// TODO: Delete all submissions for this form
	return s.Database.Collection("forms").DeleteOne(ctx, filter)
}

// CreatePipeline creates a new pipeline
func (s *Service) CreatePipeline(ctx context.Context, pipeline models.PipelineConfiguration) (*mongo.InsertOneResult, error) {
	pipeline.UpdatedAt = time.Now()
	return s.Database.Collection("pipeline_configs").InsertOne(ctx, pipeline)
}

// UpdatePipeline updates a pipeline by its ID
func (s *Service) UpdatePipeline(ctx context.Context, pipeline models.PipelineConfiguration, pipelineID primitive.ObjectID) (*mongo.UpdateResult, error) {
	pipeline.UpdatedAt = time.Now()
	update := bson.M{"$set": pipeline}
	filter := bson.M{"_id": pipelineID}
	return s.Database.Collection("pipeline_configs").UpdateOne(ctx, filter, update)
}

// GetPipeline retrieves a pipeline by its ID
func (s *Service) GetPipeline(ctx context.Context, pipelineID primitive.ObjectID) (*models.PipelineConfiguration, error) {
	var pipeline models.PipelineConfiguration
	err := s.Database.Collection("pipeline_configs").FindOne(ctx, bson.M{"_id": pipelineID}).Decode(&pipeline)
	if err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// ListPipelines retrieves pipelines based on a filter
func (s *Service) ListPipelines(ctx context.Context, filter bson.M) ([]models.PipelineConfiguration, error) {
	var pipelines []models.PipelineConfiguration

	cursor, err := s.Database.Collection("pipeline_configs").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var pipeline models.PipelineConfiguration
		if err := cursor.Decode(&pipeline); err != nil {
			return nil, err
		}

		pipelines = append(pipelines, pipeline)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	// If pipelines is null then return an empty slice instead
	if pipelines == nil {
		return []models.PipelineConfiguration{}, nil
	}

	return pipelines, nil
}

// DeletePipeline deletes a pipeline by its ID
func (s *Service) DeletePipeline(ctx context.Context, pipelineID primitive.ObjectID) (*mongo.DeleteResult, error) {
	filter := bson.M{"_id": pipelineID}
	return s.Database.Collection("pipeline_configs").DeleteOne(ctx, filter)
}