package resource

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Hazardiusoms/ADP_Project/internal/database"
	"github.com/Hazardiusoms/ADP_Project/internal/modules/auth"
)

// DeleteResourceImageHandler удаляет изображение ресурса
func DeleteResourceImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetCurrentUser(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	resourceIDStr := r.FormValue("resource_id")
	imagePath := r.FormValue("image_path")

	resourceID, err := strconv.Atoi(resourceIDStr)
	if err != nil {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("Неверный ID"), http.StatusSeeOther)
		return
	}

	// проверяем права доступа
	var createdBy int
	err = database.SafeQueryRow(
		"SELECT created_by FROM resources WHERE id = ?",
		resourceID,
	).Scan(&createdBy)

	if createdBy != user.ID && user.Role != "admin" {
		http.Redirect(w, r, "/dashboard?error="+url.QueryEscape("У вас нет прав"), http.StatusSeeOther)
		return
	}

	// получаем существующие изображения
	var existingImages string
	database.SafeQueryRow("SELECT images FROM resources WHERE id = ?", resourceID).Scan(&existingImages)
	
	var imageList []string
	if existingImages != "" {
		json.Unmarshal([]byte(existingImages), &imageList)
	}

	// удаляем изображение из списка
	newList := []string{}
	for _, img := range imageList {
		if img != imagePath {
			newList = append(newList, img)
		}
	}

	// удаляем файл с диска
	if strings.HasPrefix(imagePath, "/uploads/") {
		filePath := "." + imagePath
		os.Remove(filePath)
	}

	// обновляем базу данных
	imagesJSON, _ := json.Marshal(newList)
	database.SafeExec(
		"UPDATE resources SET images = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		string(imagesJSON), resourceID,
	)

	http.Redirect(w, r, "/resources/edit?id="+resourceIDStr+"&success="+url.QueryEscape("Изображение удалено"), http.StatusSeeOther)
}
