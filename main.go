package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"flag"
	"fmt"
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
type modutil struct {
	XMLName xml.Name `xml:"modutil"`
	Durl    string   `xml:"durl"`
	Lhash   string   `xml:"lhash"`
	Qhash   string   `xml:"qhash"`
	Mhash   string   `xml:"mhash"`
}

type config struct {
	XMLName xml.Name `xml:"config"`
	Idir    string   `xml:"idir"`
	Vers    uint8    `xml:"vers"`
}

// Constant variable declarations
const logPath = "output.log"
const confName = "config.xml"
const moduName = "modutil.xml"

// Variable declaration
var (
	// Flags for the logger setting up the verbose level
	verbose = flag.Bool("verbose", false, "print info level logs to stdout")

	lf   = &os.File{}
	edir = &nucular.TextEditor{}
	resp = &grab.Response{}

	modu = modutil{}
	conf = config{}

	dmsg, imsg, prog = "", "Integrity Check", 0

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

////////////////////
// Main Functions //
////////////////////

// Main function of the program
// Sets up the logger, reads the config files, and sets up the UI
// then all work is passed to the UI update function and buttons
func main() {
	var err error
	flag.Parse()

	lf, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Fatalf("Failed to open log file: %v", err)
	}
	defer lf.Close()

	defer logger.Init("Logger", *verbose, true, lf).Close()

	wnd := nucular.NewMasterWindow(0, "7DTD ModUtil", updatefn)
	wnd.SetStyle(style.FromTheme(style.DarkTheme, 1.0))

	// Always recreate the modutil.xml to have the most recent version
	createModConfig()
	// Read the configuration files
	readXML(moduName, modu)
	readXML(confName, conf)

	// This go function call is responsible to refresh the UI by calling wnd.changed every second
	go func() {
		for {
			time.Sleep(1 * time.Second)
			wnd.Changed()
		}
	}()

	wnd.Main()
}

// UI function
// Handles drawing and assignment of interaction
// Called through user interaction or refreshed automatically once per second
func updatefn(w *nucular.Window) {
	w.Row(10).Dynamic(1)

	w.Row(40).Ratio(0.7, 0.3)
	w.Label("7 Days to Die Mod Util", "LC")
	if w.ButtonText("Save Config") {
		writeXML(confName)
	}

	w.Row(10).Dynamic(1)

	w.Row(40).Ratio(0.4, 0.6)
	w.Label("Please set your 7DTD directory:", "LC")
	edir.Edit(w)

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

	w.Row(20).Dynamic(1)
	w.Label(dmsg, "LT")

	// If there is data being transmitted we execute the updateProgress function
	if resp.BytesPerSecond() > 0 {
		updateProgress()
	}
}

///////////////////////
// Refresh functions //
///////////////////////

// This refreshes the values for the progress display and is called within the main UI function
func updateProgress() {
	if !resp.IsComplete() {
		logger.Infof("Transferred %v / %v bytes (%.2f%%)", resp.BytesComplete(), resp.Size, 100*resp.Progress())
		prog = int(100 * resp.Progress())
	} else {
		if err := resp.Err(); err != nil {
			logger.Errorf("Download failed: %v", err)
		} else {
			logger.Infof("Download saved to ./%v", resp.Filename)
		}
	}
}

func updateTicker() {
	// TODO: Implement Ticker
}

///////////////////
// XML functions //
///////////////////

// Handles the download of the primary XML through HTTP
func createModConfig() {
	if _, err := os.Stat(moduName); err != nil {
		if os.IsNotExist(err) {
			client := grab.NewClient()
			req, _ := grab.NewRequest(".", "https://mods.netrve.net/7D2D/"+moduName)
			resp = client.Do(req)

			logger.Infof("Downloading %v...", req.URL())
			logger.Infof("  %v", resp.HTTPResponse.Status)
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

	logger.Info("Created " + confName)
}

// Reads the given XML
func readXML(filename string, target interface{}) {
	absPath, _ := filepath.Abs(filename)
	xmlFile, err := os.Open(absPath)
	if check(err) {
		switch filename {
		case confName:
			createUserConfig()
		case moduName:
			createModConfig()
		}
	}
	defer xmlFile.Close()

	data, err := ioutil.ReadAll(xmlFile)
	check(err)

	err = xml.Unmarshal([]byte(data), &target)
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

	logger.Info("Finished writing " + filename)
}

/////////////////////////
// Integrity functions //
/////////////////////////

// Generates a SHA-1 for either a given file or directory, based on the second parameter
func genHash(filein string, isdir bool) string {
	var result string

	if !isdir {
		f, err := os.Open(filein)
		check(err)
		defer f.Close()

		h := sha1.New()
		if _, err := io.Copy(h, f); err != nil {
			check(err)
		}

		result = hex.EncodeToString(h.Sum(nil))
	} else {
		hash, err := hashdir.Create(conf.Idir+"\\Mods", "sha1")
		check(err)

		result = hash
	}

	return result
}

// Integrity check for the main directories and files
// We use genHash to generate the SHA-1 hashes for the specified files
// and compare those with the hashes retrieved from the preset.xml
func checkIntegrity() bool {
	var hash1, hash2, hash3 string

	hash1 = genHash(conf.Idir+"\\Data\\Config\\Localization.txt", false)
	hash2 = genHash(conf.Idir+"\\Data\\Config\\Localization - Quest.txt", false)
	hash3 = genHash(conf.Idir+"\\Mods", true)

	logger.Infof("Hash 1: %s | Hash 2: %s | Hash 3: %s", hash1, hash2, hash3)

	pass1 := strings.EqualFold(hash1, modu.Lhash)
	pass2 := strings.EqualFold(hash2, modu.Qhash)
	pass3 := strings.EqualFold(hash3, modu.Mhash)

	logger.Infof("Pass 1: %t | Pass 2: %t | Pass 3: %t", pass1, pass2, pass3)

	imsg = fmt.Sprintf("Localization.txt: %t \nLocalization - Quest.txt: %t \nMods: %t", pass1, pass2, pass3)

	if pass1 && pass2 && pass3 {
		return true
	}

	return false
}

////////////////////////
// Download Functions //
////////////////////////

// Handles the main download for the BASE pack on which everything else is applied on top of
func downloadBase() {
	if conf.Vers < 1 {
		if _, err := os.Stat("7DTD_BASE.7z"); err != nil {
			if os.IsNotExist(err) {
				client := grab.NewClient()
				req, _ := grab.NewRequest(".", modu.Durl+"7DTD_BASE.7z")
				logger.Infof("Downloading %v...", req.URL())
				resp = client.Do(req)

				logger.Infof("  %v", resp.HTTPResponse.Status)
			}
		} else {
			prog = 100
			logger.Info("File 7DTD_BASE already exists")
		}
	}
}

func downloadUpdate() {
	if conf.Vers > 0 {
		// TODO: Implement Update Download
	}
}

///////////////////////
// Install Functions //
///////////////////////

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
