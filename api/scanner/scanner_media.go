package scanner

import (
	"context"
	"log"
	"os"
	"path"

	"github.com/photoview/photoview/api/graphql/models"
	"github.com/photoview/photoview/api/scanner/media_encoding"
	"github.com/photoview/photoview/api/scanner/scanner_cache"
	"github.com/photoview/photoview/api/scanner/scanner_task"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func getExistingMedia(tx *gorm.DB, mediaPath string) (*models.Media, error) {
	var media []*models.Media
	result := tx.Where("path_hash = ?", models.MD5Hash(mediaPath)).Find(&media)
	if result.Error != nil {
		return nil, errors.Wrap(result.Error, "scan media fetch from database")
	}
	if result.RowsAffected > 0 {
		return media[0], nil
	}
	return nil, nil
}

func createNewMedia(tx *gorm.DB, mediaPath string, albumId int, mediaTypeText models.MediaType) (*models.Media, error) {
	mediaName := path.Base(mediaPath)
	stat, err := os.Stat(mediaPath)
	if err != nil {
		return nil, err
	}

	media := models.Media{
		Title:    mediaName,
		Path:     mediaPath,
		AlbumID:  albumId,
		Type:     mediaTypeText,
		DateShot: stat.ModTime(),
	}

	if err := tx.Create(&media).Error; err != nil {
		return nil, errors.Wrap(err, "could not insert media into database")
	}

	return &media, nil
}

func ScanMedia(tx *gorm.DB, mediaPath string, albumId int, cache *scanner_cache.AlbumScannerCache) (*models.Media, bool, error) {
	media, err := getExistingMedia(tx, mediaPath)
	if err != nil {
		return nil, false, err
	}
	if media != nil {
		return media, false, nil
	}

	log.Printf("Scanning media: %s\n", mediaPath)

	mediaType, err := cache.GetMediaType(mediaPath)
	if err != nil {
		return nil, false, errors.Wrap(err, "could determine if media was photo or video")
	}

	var mediaTypeText models.MediaType
	if mediaType.IsVideo() {
		mediaTypeText = models.MediaTypeVideo
	} else {
		mediaTypeText = models.MediaTypePhoto
	}

	media, err = createNewMedia(tx, mediaPath, albumId, mediaTypeText)
	if err != nil {
		return nil, false, err
	}

	return media, true, nil
}

// ProcessSingleMedia processes a single media, might be used to reprocess media with corrupted cache
// Function waits for processing to finish before returning.
func ProcessSingleMedia(db *gorm.DB, media *models.Media) error {
	album_cache := scanner_cache.MakeAlbumCache()

	var album models.Album
	if err := db.Model(media).Association("Album").Find(&album); err != nil {
		return err
	}

	media_data := media_encoding.NewEncodeMediaData(media)

	task_context := scanner_task.NewTaskContext(context.Background(), db, &album, album_cache)
	if err := scanMedia(task_context, media, &media_data, 0, 1); err != nil {
		return errors.Wrap(err, "single media scan")
	}

	return nil
}