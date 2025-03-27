package handler

import (
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"

	tf "github.com/galeone/tensorflow/tensorflow/go"
	"github.com/labstack/echo/v4"
)

func (h *handler) ChunkedUploadHandler(c echo.Context) error {
	fileName := c.Request().Header.Get("X-File-Name")
	totalChunksStr := c.Request().Header.Get("X-Total-Chunks")
	currentChunkStr := c.Request().Header.Get("X-Current-Chunk")

	if fileName == "" || totalChunksStr == "" || currentChunkStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "missing headers",
		})
	}

	totalChunks, _ := strconv.Atoi(totalChunksStr)
	currentChunk, _ := strconv.Atoi(currentChunkStr)

	sessionInterface, _ := h.uploads.LoadOrStore(fileName, &uploadSession{
		fileName:     fileName,
		currentChunk: currentChunk,
		totalChunks:  totalChunks,
	})

	session := sessionInterface.(*uploadSession)

	session.mu.Lock()
	defer session.mu.Unlock()

	if currentChunk != session.currentChunk {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid chunk, expected %d, got %d", session.currentChunk, currentChunk),
		})
	}

	if currentChunk == 0 {
		filePath := path.Join(h.uploadDir, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to create file",
			})
		}
		session.file = file
	}

	_, err := io.Copy(session.file, c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to write chunk",
		})
	}

	session.currentChunk++

	if currentChunk == totalChunks-1 {
		session.file.Close()
		h.uploads.Delete(fileName)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "chunk upload successfully",
	})
}

func (h *handler) predictImage(path string) (any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	pixelValues := make([][][3]float32, height)
	for y := 0; y < height; y++ {
		pixelValues[y] = make([][3]float32, width)
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			pixelValues[y][x][0] = float32(r) / 65535
			pixelValues[y][x][1] = float32(g) / 65535
			pixelValues[y][x][2] = float32(b) / 65535
		}
	}

	tensor, err := tf.NewTensor(pixelValues)
	if err != nil {
		return nil, fmt.Errorf("failed to create tensor: %w", err)
	}

	output := h.model.Exec([]tf.Output{
		h.model.Op("serving_default_input_1", 0),
	}, map[tf.Output]*tf.Tensor{
		h.model.Op("serving_default_input_1", 0): tensor,
	})

	return output[0].Value(), nil
}
