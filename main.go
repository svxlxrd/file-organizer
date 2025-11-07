package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"bufio"
)

var DefaultRules = map[string]string {
    ".jpg": "Images",
    ".jpeg": "Images",
    ".png": "Images",
    ".txt": "Documents",
	".docx": "Documents",
	".doc": "Documents",
	".pdf": "Documents",
    ".mp3": "Music",
	".wav": "Music",
	".mp4": "Video",
	".avi": "Video",
	".zip": "Archives",
	".rar": "Archives",
}

type FileOrganizer struct {
	sourceDir string // директория с файлами для сортировки
	rulesMap map[string]string // правила сортировки файлов
	processedFiles int // счетчик обработанных файлов
	logFile *os.File // файл для записи операций
	statistics map[string]*FileStats // статистика по файлам
}

type FileStats struct {
	Count int // количество файлов
	TotalSize int64 // общий размер файлов
}


// NewFileOrganizer создает файл с логами и возращает новый FileOrganizer
func NewFileOrganizer(sourceDir string) *FileOrganizer {
	// открываем файл с логами: создаем файл, если его нет 
	// добавляем новые логи в конец, только для записи
	logFile, err := os.OpenFile("organizer.log", os.O_CREATE | os.O_APPEND | os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Ошибка при создании журнала логов", err)
	} // ЗАКРЫВАТЬ ФАЙЛ БУДЕМ В main()
	
	// возвращаем новый FileOrganizer
    return &FileOrganizer{
        sourceDir: sourceDir,
        rulesMap: DefaultRules,
        processedFiles: 0,
        logFile: logFile,
		statistics: make(map[string]*FileStats),
    }
}


// logSuccess логирует успешные операции
func (fo *FileOrganizer) logSuccess(message string) {
	timeLog := time.Now().Format("2006/01/02 15:04:05") // время логов
	log.SetOutput(fo.logFile) // куда будут записываться логи
	log.Printf("%v [SUCCES] %v", timeLog, message)
}

// logError логирует ошибки
func (fo *FileOrganizer) logError(message string) {
	timeLog := time.Now().Format("2006/01/02 15:04:05") // время логов
	log.SetOutput(fo.logFile) // куда будут записываться логи
	log.Printf("%v [ERROR] %v", timeLog, message)
}

//moveFile перемещает файлы из старой папки в новую, логирует все операции и возращает error
func (fo *FileOrganizer) moveFile(sourcePath, targetDir string) error {
	timeLog := time.Now().Format("2006/01/02 15:04:05") // время логов
	log.SetOutput(fo.logFile) // куда будут записываться логи


	// os.MkdirAll создает путь targetDir, если его нет
	if err := os.MkdirAll(targetDir, os.ModePerm); err != nil {
		log.Printf("%v [ERROR] Ошибка при создании папки: %v", timeLog, err)
		return err
	}


	fileName := filepath.Base(sourcePath) // имя файла
	fullPath := filepath.Join(targetDir, fileName) // полный путь: targetDir/fileName


	// перемещение файлов из sourcePath в новую директорию fullPath
	if err := os.Rename(sourcePath, fullPath); err != nil {
		log.Printf("%v [ERROR] Не удалось переместить файл %v в %v: %v", timeLog, sourcePath, fullPath, err)
		return err
	}

	log.Printf("%v [SUCCESS] Файл %s перемещён в директорию %s", timeLog, fileName, targetDir)
	return nil
}


func (fo *FileOrganizer) Organize() error {
	// находим корневую папку, где нужно провести сортировку
	root := fo.sourceDir 

	// используем filepath.Walk чтобы пройти по каждому файлу внутри root
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fo.logError(fmt.Sprintf("Ошибка с доступом к %v: %v", path, err))
			return err
		}

		// если по пути встречаем папку, пропускаем ее и смотрим дальше
		if info.IsDir() {
			return nil
		}

		// смотрим у каждого найденного файла его расширение (txt, mp4 итд)
		ext := strings.ToLower(filepath.Ext(path))

		// находим подходящую категорию
		// nameDir — результат поиска 
		// exists — логическое значение (true / false)
		nameDir, exist := fo.rulesMap[ext]
		if !exist {
			nameDir = "other" // если расширение не определено
		}

		// соединяем путь и файл для дальнейшего переноса в другую папку
		moveTo := filepath.Join(root, nameDir)


		// переносим в папку
		if err := fo.moveFile(path, moveTo); err != nil {
			fo.logError(fmt.Sprintf("Ошибка в переносе файла %v: %v", path, err))
			return err
		}

	
		// newFullPath - это новый путь к файлу, который мы будем использовать для подсчета
		newFullPath := filepath.Join(moveTo, filepath.Base(path))
		// собираем статистику с помощью os.Stat
		fileInfo, err := os.Stat(newFullPath)
		if err != nil {
			fo.logError(fmt.Sprintf("Не удалось получить информации о файле %v: %v", path, err))
		} else {
		// создаем переменную с размером файла
			size := fileInfo.Size()
		
		// stats - категория файла (image, video ...)
		// exisrs - true/false
			stats, exists := fo.statistics[nameDir]
			if !exists { // если для категории nameDir еще нет записи, то 
				// создаем новый объект в fileStats
				stats = &FileStats{}
				// сохраняем этот объект в карту под ключом nameDir
				fo.statistics[nameDir] = stats
			}

			stats.Count ++ 
			stats.TotalSize += size
		}
		
		// счетчик отсортированных файлов
		fo.processedFiles ++ 
		return nil
	})

	if err != nil {
		return err
	}

	fo.logSuccess(fmt.Sprintf("Сортировка успешно завершена. Обработано %v файлов", fo.processedFiles))
	return nil
}


// Report формирует отчет о перемещении файлов
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


// реализация пользовательского ввода
func main() {
	fmt.Println("Инструкции:")
	fmt.Println("1. Введите путь к директории")
	fmt.Println("2. Программа отсортирует файлы по категориям")
	fmt.Println("3. После сортировки будет выведен отчет")

	fmt.Println("Укажите путь к директории")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" { // если путь не указан, то используется текущая директория
		input = "./"
	}

	organizer := NewFileOrganizer(input)


	err := organizer.Organize()
	if err != nil {
		fmt.Println("Ошибка при сортировке файлов:", err)
		return
	}

	organizer.Report()
}