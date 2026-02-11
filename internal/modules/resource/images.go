package resource

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
)

// UploadResourceImages загружает изображения для ресурса
// примечание: ParseMultipartForm должен быть вызван ДО этой функции
func UploadResourceImages(resourceID int, r *http.Request) ([]string, error) {
	// проверяем если multipart форма уже распарсена
	if r.MultipartForm == nil {
		return nil, nil // нет multipart формы, нет файлов
	}

	// создаем директорию для загрузок если ее нет
	uploadDir := "./web/uploads/resources"
	os.MkdirAll(uploadDir, 0755)

	// получаем загруженные файлы
	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		return nil, nil // файлы не загружены, это нормально
	}

	var imageList []string

	// обрабатываем каждый файл
	for _, fileHeader := range files {
		// открываем файл
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer file.Close()

		// проверяем тип файла
		buffer := make([]byte, 512)
		file.Read(buffer)
		file.Seek(0, 0)
		contentType := http.DetectContentType(buffer)
		if !strings.HasPrefix(contentType, "image/") {
			continue
		}

		// генерируем уникальное имя файла
		ext := filepath.Ext(fileHeader.Filename)
		filename := strconv.Itoa(resourceID) + "_" + strconv.FormatInt(time.Now().UnixNano(), 10) + ext
		filePath := filepath.Join(uploadDir, filename)

		// создаем файл
		dst, err := os.Create(filePath)
		if err != nil {
			continue
		}
		defer dst.Close()

		// копируем файл
		io.Copy(dst, file)

		// добавляем в список изображений
		imageList = append(imageList, "/uploads/resources/"+filename)
	}

	return imageList, nil
}

// SaveResourceImages сохраняет пути к изображениям в БД
func SaveResourceImages(resourceID int, images []string) error {
	if len(images) == 0 {
		return nil
	}

	imagesJSON, err := json.Marshal(images)
	if err != nil {
		return err
	}

	_, err = database.SafeExec(
		"UPDATE resources SET images = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		string(imagesJSON), resourceID,
	)
	return err
}
