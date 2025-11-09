package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var DefaultRules = map[string]string{
	".jpg":  "Images",
	".jpeg": "Images",
	".png":  "Images",
	".txt":  "Documents",
	".docx": "Documents",
	".doc":  "Documents",
	".pdf":  "Documents",
	".mp3":  "Music",
	".wav":  "Music",
	".mp4":  "Video",
	".avi":  "Video",
	".zip":  "Archives",
	".rar":  "Archives",
}

// FileOrganizer содержит данные для сортировки файлов
type FileOrganizer struct {
	sourceDir      string
	rulesMap       map[string]string
	processedFiles int
	logFile        *os.File
	statistics     map[string]*FileStats
	logger         *log.Logger
}

// FileStats хранит статистику по файлам конкретной категории
type FileStats struct {
	Count     int
	TotalSize int64
}

// NewFileOrganizer создает новый FileOrganizer и открывает файл для логов
func NewFileOrganizer(sourceDir string) (*FileOrganizer, error) {
	logFile, err := os.OpenFile("organizer.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл логов: %v", err)
	}

	fo := &FileOrganizer{
		sourceDir:      sourceDir,
		rulesMap:       DefaultRules,
		processedFiles: 0,
		logFile:        logFile,
		statistics:     make(map[string]*FileStats),
	}
	fo.logger = log.New(logFile, "", log.LstdFlags)

	return fo, nil
}

// logSuccess логирует успешные операции
func (fo *FileOrganizer) logSuccess(message string) {
	fo.logger.Printf("[SUCCESS] %s", message)
}

// logError логирует ошибки
func (fo *FileOrganizer) logError(message string) {
	fo.logger.Printf("[ERROR] %s", message)
}

// moveFile перемещает файл в целевую директорию, добавляя timestamp при конфликте
func (fo *FileOrganizer) moveFile(sourcePath, targetDir string) error {
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		fo.logger.Printf("[ERROR] Ошибка при создании папки %s: %v", targetDir, err)
		return err
	}

	fileName := filepath.Base(sourcePath)
	fullPath := filepath.Join(targetDir, fileName)

	if _, err := os.Stat(fullPath); err == nil {
		ext := filepath.Ext(fileName)
		name := strings.TrimSuffix(fileName, ext)
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		newName := fmt.Sprintf("%s_%s%s", name, timestamp, ext)
		fullPath = filepath.Join(targetDir, newName)
		fo.logger.Printf("[INFO] Конфликт имени, новое имя: %s", newName)
	}

	if err := os.Rename(sourcePath, fullPath); err != nil {
		fo.logger.Printf("[ERROR] Не удалось переместить файл %s в %s: %v", sourcePath, fullPath, err)
		return err
	}

	fo.logger.Printf("[SUCCESS] Файл %s перемещён в %s", fileName, targetDir)
	return nil
}

// Organize выполняет сортировку файлов по категориям
func (fo *FileOrganizer) Organize() error {
	root := fo.sourceDir

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fo.logError(fmt.Sprintf("Ошибка с доступом к %v: %v", path, err))
			return err
		}

		if info.IsDir() {
			dirName := filepath.Base(path)
			for _, targetDir := range fo.rulesMap {
				if dirName == targetDir {
					return filepath.SkipDir
				}
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		nameDir, exist := fo.rulesMap[ext]
		if !exist {
			nameDir = "other"
		}

		moveTo := filepath.Join(root, nameDir)

		if err := fo.moveFile(path, moveTo); err != nil {
			fo.logError(fmt.Sprintf("Ошибка в переносе файла %v: %v", path, err))
			return err
		}

		newFullPath := filepath.Join(moveTo, filepath.Base(path))
		fileInfo, err := os.Stat(newFullPath)
		if err != nil {
			fo.logError(fmt.Sprintf("Не удалось получить информации о файле %v: %v", path, err))
		} else {
			size := fileInfo.Size()
			stats, exists := fo.statistics[nameDir]
			if !exists {
				stats = &FileStats{}
				fo.statistics[nameDir] = stats
			}
			stats.Count++
			stats.TotalSize += size
		}

		fo.processedFiles++
		return nil
	})

	if err != nil {
		return err
	}

	fo.logSuccess(fmt.Sprintf("Сортировка успешно завершена. Обработано %v файлов", fo.processedFiles))
	return nil
}

// Report выводит статистику по обработанным файлам
func (fo *FileOrganizer) Report() {
	fmt.Println("=== Отчет о перемещении файлов ===")
	fmt.Printf("Всего обработано файлов: %v\n", fo.processedFiles)

	var totalSize int
	for _, stats := range fo.statistics {
		totalSize += int(stats.TotalSize)
	}
	fmt.Printf("Общий размер: %.2f MB\n", float64(totalSize)/1024/1024)

	fmt.Println("Статистика по категориям:")
	for category, stats := range fo.statistics {
		fmt.Printf("%s:\n  - Количество файлов: %d\n  - Общий размер: %.2f MB\n", category, stats.Count, float64(stats.TotalSize)/1024/1024)
	}
}

// main выполняет ввод от пользователя и запускает сортировку
func main() {
	fmt.Println("Инструкции:")
	fmt.Println("1. Введите путь к директории")
	fmt.Println("2. Программа отсортирует файлы по категориям")
	fmt.Println("3. После сортировки будет выведен отчет")

	fmt.Println("Укажите путь к директории")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Ошибка при чтении ввода:", err)
		os.Exit(1)
	}
	input = strings.TrimSpace(input)

	if input == "" {
		input = "./"
	}

	info, err := os.Stat(input)
	if err != nil {
		fmt.Printf("Ошибка: директория '%s' недоступна или не существует: %v\n", input, err)
		os.Exit(1)
	}

	if !info.IsDir() {
		fmt.Printf("Ошибка: путь '%s' не является директорией\n", input)
		os.Exit(1)
	}

	organizer, err := NewFileOrganizer(input)
	if err != nil {
		fmt.Println("Ошибка:", err)
		os.Exit(1)
	}
	defer organizer.logFile.Close()

	if err := organizer.Organize(); err != nil {
		fmt.Println("Ошибка при сортировке:", err)
	}

	organizer.Report()
}
