
# 7DTD Modding Utility
[![Go Report Card](https://goreportcard.com/badge/github.com/Dakraid/7DTDModUtil)](https://goreportcard.com/report/github.com/Dakraid/7DTDModUtil)

This is a small work-in-progress modding utility I wrote for personal use between me and friends to make our lives easier when playing online collaborative games with mods. 

It also works as kind of an exercise in Go, which I picked up shortly before I started work on this tool.

## Features
- Simple and fast native UI
- Download and installation of given packs
- Automatic update to the most recent version
- Local file verification

## To Do
- Patch Downloads
	- Get all available patches
	- Download all patches required
- Install Routines
	- Base Installation
- Patch Routines
	- Check for base installation
	- Installation of patches in order
- Versioning
	- Used for Base and Patch routines
	- Defines what is required
- User Interface Polish
- Self-Updater
- Instruction Routines
	- Define Structure

## Dependencies
- crypto/sha1
- encoding/hex
- encoding/xml
- flag
- fmt
- image/color
- io
- io/ioutil
- os
- path/filepath
- strings
- time
- github.com/aarzilli/nucular
- github.com/aarzilli/nucular/style
- github.com/cavaliercoder/grab 
- github.com/google/logger
- github.com/sger/go-hashdir

![Go Dependency Graph](https://github.com/Dakraid/7DTDModUtil/blob/master/godepgraph.png "Go Dependency Graph")
