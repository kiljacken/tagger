package main

import (
	"code.google.com/p/go-uuid/uuid"
	"flag"
	"fmt"
	"github.com/kiljacken/tagger"
	"github.com/kiljacken/tagger/storage"
	"os"
	"strconv"
	"strings"
)

const NAME = "tagger-cli"
const VERSION = "0.0.1-alpha"
const ARG_OFFSET = 1

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
		{version, "version", "prints version information"},
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

var provider tagger.StorageProvider

func main() {
	// TODO: os.Exit(?) prohibits defers from executing. this could be bad
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	passedCmd := flag.Arg(0)
	cmd, ok := commandMap[passedCmd]
	if !ok {
		fmt.Printf("Unknown command: %s\n", passedCmd)
		usage()
		os.Exit(1)
	}

	// Setup storage provider
	prov, err := storage.NewSqliteStorage("./test.db") //":memory:")
	if err != nil {
		fmt.Printf("Error while opening storage: %s\n", err)
		os.Exit(1)
	}
	provider = prov
	defer provider.Close()

	// Execute the command
	if err = cmd.f(); err != nil {
		fmt.Printf("Error while executing command: %s\n", err)
		os.Exit(1)
	}

	// Exit with error code 0
	os.Exit(0)
}

func getFileFromArg(arg string) (tagger.File, error) {
	// If path contains the prefix 'uuid:' consider it an uuid
	if strings.HasPrefix(arg, "uuid:") {
		// Get the file matching the uuid
		return provider.GetFile(uuid.Parse(arg[5:]))
	} else {
		// Get the file matching the file
		return provider.GetFileForPath(arg)
	}
}

func ensureArgs(n int, msg string) error {
	if flag.NArg() < ARG_OFFSET+n {
		return fmt.Errorf("Expected %d arguments, got %d.\nUsage: %s", n, flag.NArg()-ARG_OFFSET, msg)
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

func version() error {
	fmt.Printf("%s v%s\n", NAME, VERSION)
	return nil
}

func addFile() error {
	// Ensure we have enough arguments
	if err := ensureArgs(1, "add [path]"); err != nil {
		return err
	}

	path := flag.Arg(ARG_OFFSET)

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

	path := flag.Arg(ARG_OFFSET)

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

	src := flag.Arg(ARG_OFFSET)
	dst := flag.Arg(ARG_OFFSET + 1)

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

	path := flag.Arg(ARG_OFFSET)
	name := flag.Arg(ARG_OFFSET + 1)

	// Get specified file
	file, err := getFileFromArg(path)
	if err != nil {
		return err
	}

	// Depending on the amount of arguments, create a value tag or a named tag
	var tag tagger.Tag
	if flag.NArg() > ARG_OFFSET+2 {
		// Parse tag value
		value, err := strconv.Atoi(flag.Arg(ARG_OFFSET + 2))
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

	path := flag.Arg(ARG_OFFSET)
	name := flag.Arg(ARG_OFFSET + 1)

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
	for i := ARG_OFFSET; i < flag.NArg(); i++ {
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
	path := flag.Arg(ARG_OFFSET)

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
