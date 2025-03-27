package handler

import (
	"os"
	"sync"

	tf "github.com/galeone/tfgo"
	"go.mongodb.org/mongo-driver/mongo"
)

type handler struct {
	client     *mongo.Client
	collection *mongo.Collection
	uploadDir  string
	uploads    sync.Map
	model      *tf.Model
}

func NewHandler(client *mongo.Client, collection *mongo.Collection, uploadDir string) *handler {
	os.Mkdir(uploadDir, os.ModePerm)

	model := tf.LoadModel("../../internal/cnn/plant_disease_cnn.keras", []string{"serve"}, nil)

	return &handler{
		client:     client,
		collection: collection,
		uploadDir:  uploadDir,
		model:      model,
	}
}

type uploadSession struct {
	fileName     string
	currentChunk int
	totalChunks  int
	file         *os.File
	mu           sync.Mutex
}
