
# HyperDragonNET Modding Utility
[![Go Report Card](https://goreportcard.com/badge/github.com/Dakraid/HDN-ModUtil)](https://goreportcard.com/report/github.com/Dakraid/HDN-ModUtil) [![Codacy Badge](https://api.codacy.com/project/badge/Grade/04f938b6805a4b3abb64b71e4ba579dd)](https://www.codacy.com/app/Dakraid/7DTDModUtil?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=Dakraid/7DTDModUtil&amp;utm_campaign=Badge_Grade)

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
- Versions
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

![Go Dependency Graph](https://github.com/Dakraid/HDN-ModUtil/blob/master/docs/godepgraph.png "Go Dependency Graph")
