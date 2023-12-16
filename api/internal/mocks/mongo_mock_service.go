package mocks

import (
	"api/internal/models"
	"api/internal/mongodb"
	"context"
	"sync"

	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
)

type MockMongoService struct {
	mock.Mock
	data  map[string]models.User // In-memory store
	mutex sync.RWMutex           // Mutex for concurrent access
}

func NewMockMongoService() *MockMongoService {
	return &MockMongoService{
		data: make(map[string]models.User),
	}
}

func (m *MockMongoService) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	user, exists := m.data[email]
	if !exists {
		return nil, mongo.ErrNoDocuments
	}
	return &user, nil
}

func (m *MockMongoService) InsertUser(ctx context.Context, user models.User) (*mongo.InsertOneResult, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.data[user.Email]; exists {
		return nil, mongodb.ErrUserAlreadyExists
	}

	m.data[user.Email] = user
	return &mongo.InsertOneResult{}, nil // Mock result
}

func (m *MockMongoService) DeleteUserByEmail(ctx context.Context, email string) (*mongo.DeleteResult, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.data[email]; !exists {
		return nil, mongo.ErrNoDocuments
	}

	delete(m.data, email)
	return &mongo.DeleteResult{}, nil // Mock result
}