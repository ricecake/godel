package cmd

import (
	"fmt"
	"os"
	"strings"
	"path/filepath"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/flosch/pongo2"
	"github.com/otiai10/copy"
)

var s *pongo2.TemplateSet

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate output file from content directory",
	Long: `This command will generate output files from the data present in the content directory.

It can reference libraries stored in the library and external directories, and it will move the conents of the
static directory into the output dir.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("generate called")

		pwd, pwdErr := os.Getwd()
		if pwdErr != nil {
			fmt.Println(pwdErr)
			os.Exit(1)
		}

		fs := pongo2.MustNewLocalFileSystemLoader("")
		s = pongo2.NewSet("test set with base directory", fs)
		s.Globals["base_directory"] = pwd
		if initErr := fs.SetBaseDir(s.Globals["base_directory"].(string)); initErr != nil {
			fmt.Printf("Got Error: %+v\n", initErr.Error())
			os.Exit(1)
		}
		err := filepath.Walk(viper.GetString("directory.content"), visit)
		if err != nil {
			fmt.Printf("Got Error: %+v\n", err.Error())
			os.Exit(1)
		}
		copyErr := copy.Copy(viper.GetString("directory.static"), filepath.Join(viper.GetString("directory.output"), viper.GetString("directory.static")))
		if copyErr != nil {
			fmt.Printf("Got Error: %+v\n", copyErr.Error())
			os.Exit(1)
		}
	},
}

func visit(path string, f os.FileInfo, err error) error {
	if !f.IsDir() {
		fmt.Printf("Visiting: %s\n", path)
		filePath := strings.TrimPrefix(path, viper.GetString("directory.content"))
		context := pongo2.Context{
			"path": filePath,
			"environment": viper.GetStringMap("environment"),
		}
		tpl, dtlErr := s.FromFile(path)
		if dtlErr != nil {
			return dtlErr
		}
		tplOut, evalErr := tpl.ExecuteBytes(context)
		if evalErr != nil {
			return evalErr
		}
		outPath := filepath.Join(viper.GetString("directory.output"), filePath)
		os.MkdirAll(filepath.Dir(outPath), os.ModePerm)
		fd, fileOpenErr := os.Create(outPath)
		if fileOpenErr != nil {
			return fileOpenErr
		}
		defer fd.Close()
		_, fileWriteErr := fd.Write(tplOut)
		if fileWriteErr != nil {
			return fileWriteErr
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
