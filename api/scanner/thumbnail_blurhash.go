package scanner

import (
	"fmt"
	"image"
	"log"
	"os"

	"github.com/buckket/go-blurhash"
	"github.com/photoview/photoview/api/graphql/models"
	"gorm.io/gorm"
)

func GenerateBlurhashes(db *gorm.DB) error {
	var results []*models.Media
	processErrors := make([]error, 0)

	query := db.Model(&models.Media{}).
		Preload("MediaURL").
		Joins("INNER JOIN media_urls ON media.id = media_urls.media_id").
		Where("blurhash IS NULL").
		Where("media_urls.purpose = 'thumbnail' OR media_urls.purpose = 'video-thumbnail'")

	err := query.FindInBatches(&results, 50, func(tx *gorm.DB, batch int) error {
		log.Printf("generating %d blurhashes", len(results))

		for i, row := range results {
			hashStr, err := generateBlurhashForRow(row)
			if err != nil {
				processErrors = append(processErrors, err)
				continue
			}

			results[i].Blurhash = &hashStr
		}

		tx.Save(results)
		return nil
	}).Error

	if err != nil {
		return err
	}

	if len(processErrors) > 0 {
		return fmt.Errorf("failed to generate %d blurhashes", len(processErrors))
	}

	return nil
}

func generateBlurhashForRow(row *models.Media) (string, error) {
	thumbnail, err := row.GetThumbnail()
	if err != nil {
		log.Printf("failed to get thumbnail for media to generate blurhash (%d): %v", row.ID, err)
		return "", err
	}

	hashStr, err := GenerateBlurhashFromThumbnail(thumbnail)
	if err != nil {
		log.Printf("failed to generate blurhash for media (%d): %v", row.ID, err)
		return "", err
	}

	return hashStr, nil
}

func GenerateBlurhashFromThumbnail(thumbnail *models.MediaURL) (string, error) {
	thumbnail_path, err := thumbnail.CachedPath()
	if err != nil {
		return "", err
	}

	imageFile, err := os.Open(thumbnail_path)
	if err != nil {
		return "", err
	}

	imageData, _, err := image.Decode(imageFile)
	if err != nil {
		return "", err
	}

	return blurhash.Encode(4, 3, imageData)
}