package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cavaliergopher/grab/v3"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	PackageName             = "Tikmeh"
	VersionInfo             = "0.1.0 (Sep 26, 2022)"
	GithubLink              = "https://github.com/mehanon/tikmeh"
	DefaultWorkingDirectory = "."
	DefaultFfmpegPath       = "ffmpeg"
	DefaultFfmpegPreset     = "faster"
)

var (
	tikwmTimeout        = 12 * time.Second
	tikwnLastReqestTime = time.Time{}
	tikwmReqestMutex    = sync.Mutex{}
)

func syncTikwmRequests() {
	tikwmReqestMutex.Lock()
	defer tikwmReqestMutex.Unlock()
	time.Sleep(tikwmTimeout - time.Since(tikwnLastReqestTime))
	tikwnLastReqestTime = time.Now()
}

// TiktokInfo there are more fields, tho I omitted unnecessary ones
type TiktokInfo struct {
	Id         string `json:"id"`
	Play       string `json:"play,omitempty"`
	Hdplay     string `json:"hdplay,omitempty"`
	CreateTime int64  `json:"create_time"`
	Author     struct {
		UniqueId string `json:"unique_id"`
	} `json:"author"`
}

type TikwmResponse struct {
	Code          int        `json:"code"`
	Msg           string     `json:"msg"`
	ProcessedTime float64    `json:"processed_time"`
	Data          TiktokInfo `json:"data,omitempty"`
}

func TikwnGetInfo(link string) (*TiktokInfo, error) {
	payload := url.Values{"url": {link}, "hd": {"1"}}
	syncTikwmRequests()
	r, err := http.PostForm("https://www.tikwm.com/api/", payload)
	if err != nil {
		return nil, err
	}
	buffer, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var resp TikwmResponse
	err = json.Unmarshal(buffer, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, errors.New(resp.Msg)
	}

	return &resp.Data, nil
}

func DownloadTiktokTikwm(link string, directory ...string) (*string, error) {
	info, err := TikwnGetInfo(link)
	if err != nil {
		return nil, err
	}

	var downloadUrl string
	if info.Hdplay != "" {
		downloadUrl = info.Hdplay
	} else if info.Play != "" {
		println("warning: tikwm couldn't find HD version, downloading how it is...")
		downloadUrl = info.Play
	} else {
		return nil, errors.New("no download links found :c")
	}

	dir := DefaultWorkingDirectory
	if len(directory) > 0 {
		dir = directory[0]
	}
	localFilename := path.Join(dir, generateFilename(info.Author.UniqueId, info.CreateTime, info.Id))

	err = Wget(downloadUrl, localFilename)
	if err != nil {
		return nil, err
	}

	return &localFilename, nil
}

type TikwmPostsResponse struct {
	Code          int     `json:"code"`
	Msg           string  `json:"msg"`
	ProcessedTime float64 `json:"processed_time"`
	Data          struct {
		Videos  []VideoInfo `json:"videos"`
		Cursor  string      `json:"cursor"`
		HasMore bool        `json:"hasMore"`
	} `json:"data,omitempty"`
}

type VideoInfo struct {
	VideoId    string `json:"video_id"`
	Play       string `json:"play"`
	Wmplay     string `json:"wmplay"`
	CreateTime int64  `json:"create_time"`
	Author     struct {
		UniqueId string `json:"unique_id"`
	} `json:"author"`
}

func TikwnGetPostsInfo(username string, cursor ...string) (*TikwmPostsResponse, error) {
	c := "0"
	if len(cursor) > 0 {
		c = cursor[0]
	}
	payload := url.Values{"unique_id": {username}, "count": {"34"}, "cursor": {c}}
	syncTikwmRequests()
	r, err := http.PostForm("https://www.tikwm.com/api/user/posts/", payload)
	if err != nil {
		return nil, err
	}
	buffer, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var resp TikwmPostsResponse
	err = json.Unmarshal(buffer, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, errors.New(resp.Msg)
	}

	return &resp, nil
}

type ProfileDownloader struct {
	Cfg parsedArgs
}

func NewProfileDownloader(args *parsedArgs) *ProfileDownloader {
	return &ProfileDownloader{Cfg: *args}
}

func (loader *ProfileDownloader) DownloadVideosHD(videos []VideoInfo, directory ...string) (isNothingNewLeft bool, err error) {
	if len(videos) == 0 {
		return false, errors.New("0 videos provided to DownloadVideosHD")
	}
	dir := DefaultWorkingDirectory
	if len(directory) != 0 {
		dir = directory[0]
	} else {
		dir = path.Join(DefaultWorkingDirectory, videos[0].Author.UniqueId)
	}
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	files := make([]string, 0)
	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".mp4") {
			files = append(files, entry.Name())
		}
	}

	for _, videoInfo := range videos {
		if isContainedInList(generateFilename(videoInfo.Author.UniqueId, videoInfo.CreateTime, videoInfo.VideoId), files) {
			if loader.Cfg.CheckAll {
				continue
			} else {
				return true, nil
			}
		}

		filename, err := DownloadTiktokTikwm(fmt.Sprintf("https://www.tiktok.com/@%s/%s", videoInfo.Author.UniqueId, videoInfo.VideoId), dir)
		if err != nil {
			return false, err
		}
		fmt.Printf("    %s", *filename)
		if loader.Cfg.Convert {
			print("    converting... ")
			_, err = convertToH264(*filename, loader.Cfg.FfmpegPath)
			if err != nil {
				log.Printf("while converting %s, an error occured: %s", *filename, err.Error())
			} else {
				print("done")
			}
		}
		println()
	}

	return false, err
}

func (loader *ProfileDownloader) DownloadProfileTikwm(username string) error {
	var dir string
	if loader.Cfg.Directory == "" {
		dir = path.Join(DefaultWorkingDirectory, strings.ToLower(strings.Trim(username, " \n@")))
	} else {
		dir = loader.Cfg.Directory
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0777)
		if err != nil {
			return err
		}
	}

	for info, err := TikwnGetPostsInfo(username, "0"); info != nil; {
		if err != nil {
			return err
		}
		isNothingNewToCheck, err := loader.DownloadVideosHD(info.Data.Videos, dir)
		if err != nil {
			return err
		}
		if isNothingNewToCheck {
			break
		}
		if !info.Data.HasMore {
			break
		}
		info, err = TikwnGetPostsInfo(username, info.Data.Cursor)
	}
	return nil
}

type parsedArgs struct {
	Help       bool
	Exit       bool
	Profile    bool
	CheckAll   bool
	Directory  string
	Convert    bool
	FfmpegPath string
	Tail       []string
}

func parseArgs(args []string) (parsed *parsedArgs) {
	parsed = &parsedArgs{
		Help:       false,
		Exit:       false,
		Profile:    false,
		CheckAll:   false,
		Directory:  "",
		Convert:    false,
		FfmpegPath: DefaultFfmpegPath,
		Tail:       make([]string, 0),
	}
	gettingDir := false
	gettingFfmpegPath := false
	for _, arg := range args {
		if isContainedInList(arg, []string{"h", "help", "-h", "--help", "-help"}) {
			parsed.Help = true
		} else if isContainedInList(arg, []string{"-p", "profile"}) {
			parsed.Profile = true
		} else if isContainedInList(arg, []string{"exit"}) {
			parsed.Exit = true
		} else if isContainedInList(arg, []string{"-d", "directory"}) {
			gettingDir = true
		} else if isContainedInList(arg, []string{"-c", "convert"}) {
			parsed.Convert = true
		} else if isContainedInList(arg, []string{"-a", "check-all"}) {
			parsed.CheckAll = true
		} else if isContainedInList(arg, []string{"-F", "ffmpeg-path"}) {
			gettingFfmpegPath = true
		} else if gettingDir {
			parsed.Directory = arg
			gettingDir = false
		} else if gettingFfmpegPath {
			parsed.FfmpegPath = arg
			gettingFfmpegPath = false
		} else {
			parsed.Tail = append(parsed.Tail, arg)
		}
	}

	if parsed.Directory == "" && !parsed.Profile {
		parsed.Directory = DefaultWorkingDirectory
	}

	return parsed
}

func printHelpMessage() {
	fmt.Printf(
		"%s %s\n"+
			"Download TikTok videos/profile in the best quality.\n"+
			"Usage: %s [OPTION]... [LINKS/USERNAMES]...\n", PackageName, VersionInfo, os.Args[0])
	if runtime.GOOS == "windows" {
		fmt.Print("Note: windows firewall could block the script from accessing the internet\n")
	}
	fmt.Printf(
		"\n"+
			"With no arguments starts in python-like interactive mode.\n"+
			"(it doesn't support argruments with spaces right now, while in command-line works fine)\n"+
			"\n"+
			"Options:\n"+
			"  -p, profile             download profiles mode\n"+
			"                            note: the script downloads videos from the most recent one,\n"+
			"                            until it meets already downloaded one, making syncing local collection easy\n"+
			"  -a, check-all           don't stop when an alredy downloaded video is met, to ensure everything is downloaded\n"+
			"  -d, directory <dir>     download directory, default <dir>=username for profiles or current for videos (created if not found)\n"+
			"  -c, convert             convert uploaded files to h264\n"+
			"  -F, ffmpeg-path <path>  path to ffmpeg, default <path>=%s (ffmpeg isn't required, unless you use 'convert')\n"+
			"\n"+
			"Examples:\n"+
			"  %s                                                     start interactive mode\n"+
			"  %s tiktok.com/@shrimpydimpy/video/7133412834960018730  simply download this tiktok\n"+
			"  %s profile @shrimpydimpy @losertron                    download all @shrimpydimpy, @losertron\n"+
			"  %s                                                     videos to ./shrimpydimpy, ./losertron accordinaly\n"+
			"  %s directory ./goddes profile @shrimpydimpy            download all @shrimpydimpy videos to ./goddes/\n"+
			"  %s profile @shrimpydimpy convert -F ./ffmpeg.exe       download and convert videos to h264\n"+
			"\n"+
			"Source files and up-to-date executables: %s\n",
		DefaultFfmpegPath, os.Args[0], os.Args[0], os.Args[0], strings.Repeat(" ", len(os.Args[0])), os.Args[0], os.Args[0], GithubLink)
	return
}

func HandleArgs(args []string) {
	arguments := parseArgs(args)

	if arguments.Help {
		printHelpMessage()
		return
	}
	if arguments.Exit {
		os.Exit(0)
	}

	if arguments.Profile {
		downloader := NewProfileDownloader(arguments)
		for _, username := range arguments.Tail {
			fmt.Printf("loading `%s` profile...\n", username)
			err := downloader.DownloadProfileTikwm(username)
			if err != nil {
				log.Printf("%v", err)
				return
			}
			println("done.")
		}
	} else {
		if _, err := os.Stat(arguments.Directory); os.IsNotExist(err) {
			err := os.Mkdir(arguments.Directory, 0777)
			if err != nil {
				log.Printf("while creating directory %s, an error occured: %s", arguments.Directory, err.Error())
				return
			}
		}
		for _, link := range arguments.Tail {
			filename, err := DownloadTiktokTikwm(link, arguments.Directory)
			if err != nil {
				log.Printf("%v", err)
				return
			}
			println(*filename)
		}
	}
}

func convertToH264(filename string, ffmpegPath ...string) (string, error) {
	var ffmpeg string
	if len(ffmpegPath) == 0 {
		ffmpeg = DefaultFfmpegPath
	} else {
		ffmpeg = ffmpegPath[0]
	}

	h264Filename := fmt.Sprintf("%s.h264.mp4", filename)
	output, err := exec.Command(ffmpeg, "-i", filename, "-vcodec", "libx264", "-acodec", "aac", "-y", "-preset", DefaultFfmpegPreset, h264Filename).Output()
	if err != nil {
		return filename, errors.New(fmt.Sprintf("while converting %s, an error occured:\n%s\n%s", filename, err.Error(), string(output)))
	}
	err = os.Rename(h264Filename, filename)
	if err != nil {
		return h264Filename, errors.New(fmt.Sprintf("while converting %s, an error occured:\n%s\n%s", filename, err.Error(), string(output)))
	}
	return filename, nil
}

func Wget(url string, filename string) error {
	_, err := grab.Get(filename, url)
	return err
}

func isContainedInList(s string, list []string) bool {
	for _, el := range list {
		if el == s {
			return true
		}
	}
	return false
}

func generateFilename(uniqueId string, timestamp int64, id string) string {
	return fmt.Sprintf(
		"%s_%s_%s.mp4",
		uniqueId,
		time.Unix(timestamp, 0).Format("2006-01-02"),
		id,
	)
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("%v", err)
			println("\nPress any button to exit...")
			_, _ = bufio.NewReader(os.Stdin).ReadByte()
		}
	}()

	if len(os.Args) > 1 { // with console args
		HandleArgs(os.Args[1:])
	} else { // interactive mode
		fmt.Printf("%s %s Sources and up-to-date executables: %s\n"+
			"Enter 'help' to get help message.\n", PackageName, VersionInfo, GithubLink)
		reader := bufio.NewReader(os.Stdin)
		for {
			print(">>> ")
			input, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf(err.Error())
			}
			input = strings.Trim(input, " \n\t")
			if input == "" {
				println("see you next time (exiting in 5 sec)")
				time.Sleep(time.Second * 5)
				os.Exit(0)
			}
			HandleArgs(strings.Split(input, " "))
		}
	}
}
