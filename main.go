package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"flag"
	"fmt"
	"image/color"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aarzilli/nucular"
	"github.com/aarzilli/nucular/style"
	"github.com/cavaliercoder/grab"
	"github.com/google/logger"
	"github.com/sger/go-hashdir"
)

// Type definition for the XML data
// Application specific configuration
type modutil struct {
	XMLName xml.Name `xml:"modutil"`
	// URL to the server used to download from
	Server string `xml:"server"`
	// All hashes to be used to identify mod files
	Hashes []hashes `xml:"hashes"`
}

type hashes struct {
	XMLName xml.Name `xml:"hashes"`
	// SHA-1 hash itself
	Hash string `xml:"hash>Value"`
	// Target that the hash is for, can be dir or file
	Target string `xml:"hash>Target"`
}

// User specific configuration
type config struct {
	XMLName xml.Name `xml:"config"`
	// Unique game id, used for selecting what game to operate on
	Guid string `xml:"guid"`
	// Install directory for the game in question
	Idir string `xml:"idir"`
	// Version of the currently installed pack
	// 0 implies no installed pack, 1 when base pack is applied, 1+n for patches
	Vers uint8 `xml:"vers"`
}

// Filename declaration
const (
	logPath  = "output.log"
	confName = "config.xml"
	moduName = "modutil.xml"
)

// Color declaration
var (
	white  = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	green  = color.RGBA{G: 255, A: 255}
	yellow = color.RGBA{R: 255, G: 255, A: 255}
	orange = color.RGBA{R: 255, G: 127, A: 255}
	red    = color.RGBA{R: 255, A: 255}
	dcolor = white
	dmsg   []string
)

// Variable declaration
var (
	// Flags for the logger setting up the verbose level
	verbose = flag.Bool("verbose", false, "print info level logs to stdout")

	edir = &nucular.TextEditor{}
	resp = &grab.Response{}

	modu = modutil{}
	conf = config{}

	imsg, prog, dfin = "Integrity Check", 0, false
)

// Error check function
// Mainly for cleanliness
func check(e error) (haserr bool) {
	if e != nil {
		logger.Errorf(e.Error())
		return true
	}
	return false
}

func clog(typein int8, message string) {
	switch typein {
	case 0:
		logger.Info(message)
		dcolor = green
		dmsg = append(dmsg, "[INFO] "+message)
	case 1:
		logger.Warning(message)
		dcolor = yellow
		dmsg = append(dmsg, "[WARNING] "+message)
	case 2:
		logger.Error(message)
		dcolor = orange
		dmsg = append(dmsg, "[ERROR] "+message)
	case 3:
		logger.Fatal(message)
		dcolor = red
		dmsg = append(dmsg, "[FATAL] "+message)
	}
	if len(dmsg) > 4 {
		dmsg[0] = ""
		dmsg = dmsg[1:]
	}
}

// //////////////////
// Main Functions //
// //////////////////

// Main function of the program
// Sets up the logger, reads the config files, and sets up the UI
// then all work is passed to the UI update function and buttons
func main() {
	flag.Parse()

	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0660)
	if err != nil {
		clog(3, fmt.Sprintf("Failed to open log file: %v", err))
	}
	defer lf.Close()

	defer logger.Init("Logger", *verbose, true, lf).Close()

	wnd := nucular.NewMasterWindow(0, "HDN ModUtil", updatefn)
	wnd.SetStyle(style.FromTheme(style.DarkTheme, 1.0))

	// Always recreate the modutil.xml to have the most recent version
	createModConfig()
	// Read the configuration files
	readXML(moduName)
	readXML(confName)

	// This go function call is responsible to refresh the UI by calling wnd.changed every second
	go func() {
		for {
			time.Sleep(1 * time.Second)
			wnd.Changed()
		}
	}()

	clog(0, "Welcome to HDN ModUtil")
	wnd.Main()
}

// UI function
// Handles drawing and assignment of interaction
// Called through user interaction or refreshed automatically once per second
func updatefn(w *nucular.Window) {
	w.Row(10).Dynamic(1)

	w.Row(40).Ratio(0.7, 0.3)
	w.Label("HyperDragonNET Modding Utility", "LC")
	if w.ButtonText("Save Config") {
		writeXML(confName)
	}

	w.Row(10).Dynamic(1)

	w.Row(40).Ratio(0.3, 0.6, 0.1)
	w.Label("Set your game directory:", "LC")
	edir.Edit(w)
	edir.Flags = nucular.EditClipboard | nucular.EditSigEnter
	if w.ButtonText("Set") {
		setInstallDir()
	}

	w.Row(20).Dynamic(1)

	w.Row(40).Dynamic(1)
	if w.ButtonText("Check Integrity") {
		checkIntegrity()
	}
	w.Row(50).Dynamic(1)
	w.Label(imsg, "CT")

	w.Row(20).Dynamic(1)

	w.Row(30).Dynamic(2)
	if w.ButtonText("Download Base") {
		downloadBase()
	}
	if w.ButtonText("Download Update") {
		downloadUpdate()
	}

	w.Row(30).Dynamic(2)
	if w.ButtonText("Install Base") {
		installBase()
	}
	if w.ButtonText("Install Update") {
		installUpdate()
	}

	w.Row(20).Dynamic(1)

	w.Row(20).Dynamic(1)
	w.Progress(&prog, 100, false)

	w.Row(0).Dynamic(1)
	w.LabelColored(strings.Join(dmsg, "\n"), "LT", dcolor)

	// If there is data being transmitted we execute the updateProgress function
	if resp.BytesPerSecond() > 0 {
		updateProgress()
	}
}

// /////////////////////
// Refresh functions //
// /////////////////////

// This refreshes the values for the progress display and is called within the main UI function
func updateProgress() {
	if !resp.IsComplete() {
		clog(0, fmt.Sprintf("Transferred %v / %v bytes (%.2f%%)", resp.BytesComplete(), resp.Size, 100*resp.Progress()))
		prog = int(100 * resp.Progress())
	} else {
		if err := resp.Err(); err != nil {
			clog(2, fmt.Sprintf("Download failed: %v", err))
		}
		if !dfin {
			dfin = true
			clog(0,"Download finished:"+conf.Guid+"_BASE.7z")
		}
	}
}

// /////////////////
// XML functions //
// /////////////////

// Handles the download of the primary XML through HTTP
func createModConfig() {
	if _, err := os.Stat(moduName); err != nil {
		if os.IsNotExist(err) {
			client := grab.NewClient()
			req, _ := grab.NewRequest(".", "https://mods.netrve.net/"+conf.Guid+"/"+moduName)
			resp = client.Do(req)

			clog(0, fmt.Sprintf("Downloading %v...", req.URL()))
			clog(0, fmt.Sprintf("  %v", resp.HTTPResponse.Status))
		}
	}
}

// Create an empty config.xml
func createUserConfig() {
	output, err := xml.MarshalIndent(conf, "  ", "    ")
	check(err)

	absPath, _ := filepath.Abs(confName)
	err = ioutil.WriteFile(absPath, output, 0644)
	check(err)

	clog(0, fmt.Sprintf("Created "+confName))
}

// Reads the given XML
func readXML(filename string) {
	absPath, _ := filepath.Abs(filename)
	xmlFile, err := os.Open(absPath)
	if check(err) {
		switch filename {
		case confName:
			createUserConfig()
		case moduName:
			createModConfig()
		default:
			logger.Fatal("Unrecognized input file")
		}
	}
	defer xmlFile.Close()

	data, err := ioutil.ReadAll(xmlFile)
	check(err)

	switch filename {
	case confName:
		err = xml.Unmarshal([]byte(data), &conf)
	case moduName:
		err = xml.Unmarshal([]byte(data), &modu)
	default:
		logger.Fatal("Unrecognized input file")
	}
	check(err)

	edir.InsertMode = true
	edir.Cursor = 0
	edir.Text([]rune(conf.Idir))
}

// Writes to the given XML
func writeXML(filename string) {
	output, err := xml.MarshalIndent(conf, "  ", "    ")
	check(err)

	absPath, _ := filepath.Abs(filename)
	err = ioutil.WriteFile(absPath, output, 0644)
	check(err)

	clog(0, fmt.Sprintf("Finished writing "+filename))
}

func setInstallDir() {
	conf.Idir = string(edir.Buffer)
	clog(0, fmt.Sprintf("Install directory set, don't forget to save!"))
}

func setInstallVers(input string) {
	conf.Idir = input
	clog(0, "Version set to "+input)
}

// ///////////////////////
// Integrity functions //
// ///////////////////////

// Used for determining target type
func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil{
		return false, err
	}
	return fileInfo.IsDir(), err
}

// Generates a SHA-1 for either a given file or directory, based on the second parameter
func genHash(target string) string {
	var result string

	isdir, err := isDirectory(conf.Idir+target)
	check(err)

	if !isdir {
		f, err := os.Open(conf.Idir+target)
		check(err)
		defer f.Close()

		h := sha1.New()
		if _, err := io.Copy(h, f); err != nil {
			check(err)
		}

		result = hex.EncodeToString(h.Sum(nil))
	} else {
		hash, err := hashdir.Create(conf.Idir+target, "sha1")
		check(err)

		result = hash
	}

	return result
}

// Integrity check for the main directories and files
// We use genHash to generate the SHA-1 hashes for the specified files
// and compare those with the hashes retrieved from the preset.xml
func checkIntegrity() bool {

	if len(conf.Idir) > 0 {



		// imsg = fmt.Sprintf("Localization.txt: %t \nLocalization - Quest.txt: %t \nMods: %t", pass1, pass2, pass3)

		return false
	} else {
		clog(1, "Install directory is not set")

		return false
	}
}

// //////////////////////
// Download Functions //
// //////////////////////

// Handles the main download for the BASE pack on which everything else is applied on top of
func downloadBase() {
	if conf.Vers < 1 {
		if _, err := os.Stat(conf.Guid+"_BASE.7z"); err != nil {
			if os.IsNotExist(err) {
				client := grab.NewClient()
				req, _ := grab.NewRequest(".", modu.Server+conf.Guid+"/"+conf.Guid+"_BASE.zip")
				clog(0, fmt.Sprintf("Downloading %v...", req.URL()))
				resp = client.Do(req)

				clog(0, fmt.Sprintf("  %v", resp.HTTPResponse.Status))

				dfin = false
			}
		} else {
			prog = 100
			clog(0, "File already exists")
		}
	}
}

func downloadUpdate() {
	if conf.Vers > 0 {
		// TODO: Implement Update Download
	}
}

// /////////////////////
// Install Functions //
// /////////////////////

func installBase() {
	if conf.Vers < 1 {
		// TODO: Implement Base Install
	}
}

func installUpdate() {
	if conf.Vers > 0 {
		// TODO: Implement Update Install
	}
}
