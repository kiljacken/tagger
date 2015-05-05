package main

import (
	"flag"
	"fmt"
	"github.com/kiljacken/go-uuid/uuid"
	"github.com/kiljacken/tagger"
	"github.com/kiljacken/tagger/storage"
	"os"
	"strconv"
	"strings"
)

const version = "v0.0.1"
const argOffset = 1

type command struct {
	f    func() error
	name string
	desc string
}

var commands []command
var commandMap map[string]command

func init() {
	commands = []command{
		{usage, "help", "prints a helpful usage message"},
		{versionCmd, "version", "prints version information"},
		// File manipulation
		{addFile, "add", "adds a file to the tag database"},
		{removeFile, "remove", "removes a file from the tag database"},
		{moveFile, "move", "moves a file to a new location"},
		// Tag manipulation
		{setTag, "set", "sets a tag on a file"},
		{unsetTag, "unset", "unsets a tag on a file"},
		// Querying
		{match, "match", "find files matching filter"},
		{get, "get", "gets the tags on a file"},
		{files, "files", "gets all files in database"},
	}

	commandMap = map[string]command{}
	for _, cmd := range commands {
		commandMap[cmd.name] = cmd
	}
}

var config = flag.String("config", DefaultPath(), "specifiy a configuration file")

var provider tagger.StorageProvider
var configuration *Configuration

func main() {
	if ok := realMain(); !ok {
		os.Exit(1)
	}
	os.Exit(0)
}

func realMain() bool {
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		return false
	}

	// Load configuration
	configuration = DefaultConfiguration()
	if _, err := os.Stat(*config); !os.IsNotExist(err) {
		f, err := os.Open(*config)
		if err != nil {
			fmt.Printf("Error while reading config: %s\n", err)
			return false
		}

		if err := configuration.Read(f); err != nil {
			fmt.Printf("Error while reading config: %s\n", err)
			return false
		}

		_ = f.Close()
	}

	// Parse command
	passedCmd := flag.Arg(0)
	cmd, ok := commandMap[passedCmd]
	if !ok {
		fmt.Printf("Unknown command: %s\n", passedCmd)
		usage()
		return false
	}

	// Setup storage provider
	prov, err := storage.NewSqliteStorage(configuration.DatabasePath())
	if err != nil {
		fmt.Printf("Error while opening storage: %s\n", err)
		return false
	}
	provider = prov
	defer provider.Close()

	// Execute the command
	if err = cmd.f(); err != nil {
		fmt.Printf("Error while executing command: %s\n", err)
		return false
	}

	// Save configuration
	f, err := os.Create(*config)
	if err != nil {
		fmt.Printf("Error while saving configuration: %s\n", err)
		return false
	}
	defer f.Close()

	err = configuration.Write(f)
	if err != nil {
		fmt.Printf("Error while saving configuration: %s\n", err)
		return false
	}

	// Exit with error code 0
	return true
}

func getFileFromArg(arg string) (tagger.File, error) {
	// If path contains the prefix 'uuid:' consider it an uuid
	if strings.HasPrefix(arg, "uuid:") {
		// Get the file matching the uuid
		return provider.GetFile(uuid.Parse(arg[5:]))
	}

	// Get the file matching the file
	return provider.GetFileForPath(arg)
}

func ensureArgs(n int, msg string) error {
	if flag.NArg() < argOffset+n {
		return fmt.Errorf("Expected %d arguments, got %d.\nUsage: %s", n, flag.NArg()-argOffset, msg)
	}
	return nil
}

func usage() error {
	fmt.Printf("Usage: tagger-cli [command] <arguments>\n")
	fmt.Printf("\n")
	fmt.Printf("Available commands:\n")
	for _, cmd := range commands {
		fmt.Printf("  %s: %s\n", cmd.name, cmd.desc)
	}

	return nil
}

func versionCmd() error {
	fmt.Printf("tagger-cli %s\n", version)
	return nil
}

func addFile() error {
	// Ensure we have enough arguments
	if err := ensureArgs(1, "add [path]"); err != nil {
		return err
	}

	path := flag.Arg(argOffset)

	// Create the new file
	file := tagger.NewFile(uuid.NewUUID(), path)

	// Update the file, an return if an error occurs
	err := provider.UpdateFile(file, []tagger.Tag{})
	if err != nil {
		return err
	}

	// Print the new uuid to the user
	fmt.Printf("%s\n", file.UUID())

	return nil
}

func removeFile() error {
	// Ensure we have enough arguments
	if err := ensureArgs(1, "remove [path]"); err != nil {
		return err
	}

	path := flag.Arg(argOffset)

	// Get the file matching the supplied argument
	file, err := getFileFromArg(path)
	if err != nil {
		return err
	}

	// Remove the file and return the error value
	return provider.RemoveFile(file)
}

func moveFile() error {
	// Ensure we have enough arguments
	if err := ensureArgs(2, "move [source] [destination]"); err != nil {
		return err
	}

	src := flag.Arg(argOffset)
	dst := flag.Arg(argOffset + 1)

	// Get the file matching the supplied argument
	file, err := getFileFromArg(src)
	if err != nil {
		return err
	}

	// Get the tags of the file
	tags, err := provider.GetTags(file)
	if err != nil {
		return err
	}

	// Update the file path
	file = tagger.NewFile(file.UUID(), dst)

	// Update the file and return the error value
	return provider.UpdateFile(file, tags)
}

func setTag() error {
	// Ensure we have enough arguments
	if err := ensureArgs(2, "set [path] [tag] (value)"); err != nil {
		return err
	}

	path := flag.Arg(argOffset)
	name := flag.Arg(argOffset + 1)

	// Get specified file
	file, err := getFileFromArg(path)
	if err != nil {
		return err
	}

	// Depending on the amount of arguments, create a value tag or a named tag
	var tag tagger.Tag
	if flag.NArg() > argOffset+2 {
		// Parse tag value
		value, err := strconv.Atoi(flag.Arg(argOffset + 2))
		if err != nil {
			return tagger.ErrInvalidValue
		}

		// Create a value tag
		tag = tagger.NewValueTag(name, int(value))
	} else {
		// Create a named tag
		tag = tagger.NewNamedTag(name)
	}

	// Update the tag and return any errors
	return provider.UpdateTag(file, tag)
}

func unsetTag() error {
	// Ensure we have enough arguments
	if err := ensureArgs(2, "unset [path] [tag]"); err != nil {
		return err
	}

	path := flag.Arg(argOffset)
	name := flag.Arg(argOffset + 1)

	// Get specified file
	file, err := getFileFromArg(path)
	if err != nil {
		return err
	}

	tag := tagger.NewNamedTag(name)

	// Update the tag and return any errors
	return provider.RemoveTag(file, tag)
}

func match() error {
	if err := ensureArgs(1, "match [filter]"); err != nil {
		return err
	}

	// Stich filter together from arguments for user convinience
	arg := ""
	for i := argOffset; i < flag.NArg(); i++ {
		arg = fmt.Sprintf("%s %s", arg, flag.Arg(i))
	}

	// Parse the filter
	r := strings.NewReader(arg)
	filter, err := tagger.ParseFilter(r)
	if err != nil {
		return err
	}

	// Get all files matching the filter
	files, err := provider.GetMatchingFiles(filter)
	if err != nil {
		return err
	}

	// Print all matched files
	for _, file := range files {
		fmt.Printf("%s %s\n", file.UUID(), file.Path())
	}

	return nil
}

func get() error {
	if err := ensureArgs(1, "get [file]"); err != nil {
		return err
	}
	path := flag.Arg(argOffset)

	// Get the provided file
	file, err := getFileFromArg(path)
	if err != nil {
		return nil
	}

	// Get the tags for the file
	tags, err := provider.GetTags(file)
	if err != nil {
		return err
	}

	// Loop through each tag and print it out
	for _, tag := range tags {
		if tag.HasValue() {
			fmt.Printf("%s=%d ", tag.Name(), tag.Value())
		} else {
			fmt.Printf("%s ", tag.Name())
		}
	}
	fmt.Printf("\n")

	return nil
}

func files() error {
	// Get the list of all files
	files, err := provider.GetAllFiles()
	if err != nil {
		return err
	}

	// Loop through each file and print their UUID and path
	for _, file := range files {
		fmt.Printf("%s %s\n", file.UUID(), file.Path())
	}

	return nil
}
