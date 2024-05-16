package main

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // Importar decodificadores de imagen
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go"
	"github.com/cloudinary/cloudinary-go/api/admin"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
		return
	}

	// Configurar Viper para leer variables de entorno
	viper.AutomaticEnv()

	// Descargar imágenes de Cloudinary
	// downloadImagesFromCloudinary()

	// Definir la carpeta que contiene las imágenes
	dir := fmt.Sprintf("./%s/subidas/", viper.GetString("APP_FOLDER"))
	outputDir := "./converted"
	// Create the output directory if it doesn't exist
	err = os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating output directory:", err)
		return
	}
	// Leer los archivos de la carpeta
	files, err := readImageFiles(dir)
	if err != nil {
		fmt.Println("Error reading files:", err)
		return
	}

	fmt.Println("Files to check:", len(files))

	// Imprimir los nombres de los archivos
	for _, file := range files {
		if strings.Contains(file, "png") {
			// fmt.Println(file)
			err := convertToJPG(file, outputDir)
			if err != nil {
				fmt.Println("Error converting file:", file, err)
			} else {
				fmt.Println("Converted file:", file)
				// Remove the original file
				err = os.Remove(file)
				if err != nil {
					fmt.Println("error removing original file: %w", err)
					return
				}
			}
		}
	}
}

// Read image files from a directory
func readImageFiles(dir string) ([]string, error) {
	var imageFiles []string

	// Check if a file is an image
	isImageFile := func(name string) bool {
		extensions := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp"}
		for _, ext := range extensions {
			if strings.HasSuffix(strings.ToLower(name), ext) {
				return true
			}
		}
		return false
	}

	// Walk the directory
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isImageFile(info.Name()) {
			imageFiles = append(imageFiles, path)
		}
		return nil
	})

	return imageFiles, err
}

// Convert an image to JPG
func convertToJPG(inputPath, outputDir string) error {
	// Open the image file
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("error decoding file: %w", err)
	}

	// Verify the format and proceed only if it's a known image format
	switch format {
	case "jpeg", "png", "gif", "bmp", "tiff":
		// valid formats
	default:
		return fmt.Errorf("unsupported image format: %s", format)
	}

	// Convert the image to JPEG
	outputPath := filepath.Join(outputDir, strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))+".jpg")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outputFile.Close()

	// Set JPEG options
	var opts jpeg.Options
	opts.Quality = 90

	// Encode the image to JPEG format
	err = jpeg.Encode(outputFile, img, &opts)
	if err != nil {
		return fmt.Errorf("error encoding to jpeg: %w", err)
	}
	// Move the JPEG file to the final output directory
	finalOutputPath := filepath.Join(outputDir, filepath.Base(outputPath))
	err = os.Rename(outputPath, finalOutputPath)
	if err != nil {
		return fmt.Errorf("error moving file to final output directory: %w", err)
	}

	// Remove the original file
	err = os.Remove(inputPath)
	if err != nil {
		return fmt.Errorf("error removing original file: %w", err)
	}

	return nil
}

func downloadImagesFromCloudinary() {
	// Configurar la API de Cloudinary
	ctx := context.Background()
	cl, err := cloudinary.NewFromParams(
		viper.GetString("APP_CLOUDINARY"),
		viper.GetString("APP_CLOUDINARY_KEY"),
		viper.GetString("APP_CLOUDINARY_SECRET"),
	)
	if err != nil {
		fmt.Println("Error al configurar la API de Cloudinary:", err)
		return
	}
	search := true
	counter := 0
	init := ""
	for {
		if !search || counter >= 200 {
			break
		}

		// Obtener la lista de recursos (fotos) en la raíz de la cuenta
		resp, err := cl.Admin.Assets(ctx, admin.AssetsParams{
			MaxResults: 5,
			NextCursor: init,
		})
		if err != nil {
			fmt.Println("Error al obtener la lista de recursos:", err)
			return
		}
		for _, asset := range resp.Assets {
			outputFile := fmt.Sprintf("%s/%v.%v", viper.GetString("APP_FOLDER"), asset.PublicID, asset.Format)
			fmt.Println("Descargando:", outputFile)
			// Verificar si el archivo ya existe en el sistema de archivos
			if _, err := os.Stat(outputFile); err == nil {
				fmt.Println("El archivo ya existe:", outputFile)
				continue
			}
			// Realiza la solicitud HTTP GET para obtener la imagen
			response, err := http.Get(asset.URL)
			if err != nil {
				fmt.Println("Error al obtener la imagen:", err)
				return
			}
			defer response.Body.Close()

			// Crea un archivo en el sistema de archivos para escribir la imagen descargada
			file, err := os.Create(outputFile)
			if err != nil {
				fmt.Println("Error al crear el archivo:", err)
				return
			}
			defer file.Close()

			// Copia el cuerpo de la respuesta HTTP al archivo
			_, err = io.Copy(file, response.Body)
			if err != nil {
				fmt.Println("Error al escribir el archivo:", err)
				return
			}

			fmt.Println("Imagen descargada correctamente.")
			counter++
		}
		init = resp.NextCursor
		fmt.Println(
			resp.NextCursor,
		)
	}
}
