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

type preset struct {
	XMLName xml.Name `xml:"preset"`
	Durl    string   `xml:"durl"`
	Lhash   string   `xml:"lhash"`
	Qhash   string   `xml:"qhash"`
	Mhash   string   `xml:"mhash"`
}

type user struct {
	XMLName xml.Name `xml:"user"`
	Idir    string   `xml:"idir"`
	Vers    uint8    `xml:"vers"`
}

type modutil struct {
	XMLName xml.Name `xml:"modutil"`
	Preset  preset   `xml:"preset"`
	User    user     `xml:"user"`
}

const logPath = "output.log"

var (
	inDir                      = &nucular.TextEditor{}
	v                          = modutil{}
	verbose                    = flag.Bool("verbose", false, "print info level logs to stdout")
	isLoaded, imsg, dmsg, prog = false, "Integrity Check", "Welcome", 0
	green                      = color.RGBA{100, 255, 100, 255}
	red                        = color.RGBA{255, 100, 100, 255}
	white                      = color.RGBA{255, 255, 255, 255}
	dcolor                     = white
	resp                       = &grab.Response{}
)

func check(e error) {
	if e != nil {
		logger.Fatalf(e.Error())
	}
}

func main() {
	flag.Parse()

	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Fatalf("Failed to open log file: %v", err)
	}
	defer lf.Close()

	defer logger.Init("Logger", *verbose, true, lf).Close()

	wnd := nucular.NewMasterWindow(0, "7DTD ModUtil", updatefn)
	wnd.SetStyle(style.FromTheme(style.DarkTheme, 1.0))
	readConfig("preset.xml")
	readConfig("user.xml")
	go func() {
		for {
			time.Sleep(1 * time.Second)
			wnd.Changed()
		}
	}()
	wnd.Main()
}

func updatefn(w *nucular.Window) {
	w.Row(10).Dynamic(1)

	w.Row(40).Ratio(0.7, 0.3)
	w.Label("7 Days to Die Mod Util", "LC")
	if w.ButtonText("Save Config") {
		writeConfig()
		dmsg = "Config written"
		dcolor = green
	}

	w.Row(10).Dynamic(1)

	w.Row(40).Ratio(0.4, 0.6)
	w.Label("Please set your 7DTD directory:", "LC")
	inDir.Edit(w)

	w.Row(20).Dynamic(1)

	w.Row(40).Dynamic(1)
	if w.ButtonText("Check Integrity") {
		if checkIntegrity() {
			dmsg = "Integrity check passed"
			dcolor = green
		} else {
			dmsg = "Integrity check failed"
			dcolor = red
		}
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
	w.LabelColored(dmsg, "LT", dcolor)

	if resp.BytesPerSecond() > 0 {
		updateProgress()
	}
}

func downloadPreset() {
	if _, err := os.Stat("preset.xml"); err != nil {
		if os.IsNotExist(err) {
			client := grab.NewClient()
			req, _ := grab.NewRequest(".", "https://mods.netrve.net/7D2D/preset.xml")
			logger.Infof("Downloading %v...", req.URL())
			dmsg = fmt.Sprintf("Downloading %v...", req.URL())
			dcolor = green
			resp = client.Do(req)
			logger.Infof("  %v", resp.HTTPResponse.Status)
		}
	} else {
		prog = 100
		dmsg = "Preset.xml already downloaded"
		dcolor = green
	}
}

func readConfig(filename string) {
	downloadPreset()
	absPath, _ := filepath.Abs(filename)
	xmlFile, err := os.Open(absPath)
	check(err)
	defer xmlFile.Close()

	data, err := ioutil.ReadAll(xmlFile)
	check(err)

	err = xml.Unmarshal([]byte(data), &v)
	check(err)

	inDir.InsertMode = true
	inDir.Cursor = 0
	inDir.Text([]rune(v.User.Idir))

	isLoaded = true
}

func writeConfig() {
	if isLoaded {
		output, err := xml.MarshalIndent(v, "  ", "    ")
		check(err)

		absPath, _ := filepath.Abs("user.xml")
		err = ioutil.WriteFile(absPath, output, 0644)
		check(err)
		logger.Info("Finished writing user.xml")
	} else {
		logger.Errorf("Please load user.xml before attempting to write")
	}
}

func genHash(filedir string, isdir bool) string {
	if !isdir {
		f, err := os.Open(filedir)
		check(err)
		defer f.Close()

		h := sha1.New()
		if _, err := io.Copy(h, f); err != nil {
			check(err)
		}

		return hex.EncodeToString(h.Sum(nil))
	} else {
		hash, err := hashdir.Create(v.User.Idir+"\\Mods", "sha1")
		check(err)
		return hash
	}
}

func checkIntegrity() bool {
	var hash1, hash2, hash3 string

	hash1 = genHash(v.User.Idir+"\\Data\\Config\\Localization.txt", false)
	hash2 = genHash(v.User.Idir+"\\Data\\Config\\Localization - Quest.txt", false)
	hash3 = genHash(v.User.Idir+"\\Mods", true)

	logger.Infof("Hash 1: %s | Hash 2: %s | Hash 3: %s", hash1, hash2, hash3)

	pass1 := strings.EqualFold(hash1, v.Preset.Lhash)
	pass2 := strings.EqualFold(hash2, v.Preset.Qhash)
	pass3 := strings.EqualFold(hash3, v.Preset.Mhash)

	logger.Infof("Pass 1: %t | Pass 2: %t | Pass 3: %t", pass1, pass2, pass3)

	imsg = fmt.Sprintf("Localization.txt: %t \nLocalization - Quest.txt: %t \nMods: %t", pass1, pass2, pass3)

	if pass1 && pass2 && pass3 {
		return true
	}

	return false
}

func updateProgress() {
	if !resp.IsComplete() {
		dmsg = fmt.Sprintf("Transferred %v / %v bytes (%.2f%%)", resp.BytesComplete(), resp.Size, 100*resp.Progress())
		logger.Infof(dmsg)
		prog = int(100 * resp.Progress())
	} else {
		if err := resp.Err(); err != nil {
			logger.Errorf("Download failed: %v", err)
			dmsg = fmt.Sprintf("Download failed: %v", err)
			dcolor = red
		} else {
			dmsg = fmt.Sprintf("Download saved to ./%v", resp.Filename)
			dcolor = green
		}
		logger.Infof(dmsg)
	}
}

func downloadBase() {
	if v.User.Vers < 1 {
		if _, err := os.Stat("7DTD_BASE.7z"); err != nil {
			if os.IsNotExist(err) {
				client := grab.NewClient()
				req, _ := grab.NewRequest(".", v.Preset.Durl+"7DTD_BASE.7z")
				logger.Infof("Downloading %v...", req.URL())
				dmsg = fmt.Sprintf("Downloading %v...", req.URL())
				dcolor = green
				resp = client.Do(req)
				logger.Infof("  %v", resp.HTTPResponse.Status)
			}
		} else {
			prog = 100
			dmsg = "7DTD_BASE.7z already downloaded"
			dcolor = green
		}
	}
}

func downloadUpdate() {
	if v.User.Vers > 0 {
		// TODO: Implement Update Download
	}
}

func installBase() {
	if v.User.Vers < 1 {
		// TODO: Implement Base Install
	}
}

func installUpdate() {
	if v.User.Vers > 0 {
		// TODO: Implement Update Install
	}
}
