package cmd

import (
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

		var files []string
		gatherErr := filepath.Walk(viper.GetString("directory.content"), func(path string, f os.FileInfo, err error) error {
			if !f.IsDir() {
				files = append(files, path)
			}
			return nil
		})
		if gatherErr != nil {
			fmt.Printf("Got Error: %+v\n", gatherErr.Error())
			os.Exit(1)
		}

		sort.Sort(fileList(files))

		fs := pongo2.MustNewLocalFileSystemLoader("")
		s = pongo2.NewSet("test set with base directory", fs)
		s.Globals["base_directory"] = pwd
		if initErr := fs.SetBaseDir(s.Globals["base_directory"].(string)); initErr != nil {
			fmt.Printf("Got Error: %+v\n", initErr.Error())
			os.Exit(1)
		}

		for _, file := range files {
			fmt.Printf("Visiting: %s\n", file)
			filePath := strings.TrimPrefix(file, viper.GetString("directory.content"))
			context := pongo2.Context{
				"dir":         filepath.Dir(filePath),
				"path":        filePath,
				"environment": viper.GetStringMap("environment"),
			}
			tpl, dtlErr := s.FromFile(file)
			if dtlErr != nil {
				fmt.Printf("Got Error: %+v\n", dtlErr.Error())
				os.Exit(1)
			}
			tplOut, evalErr := tpl.ExecuteBytes(context)
			if evalErr != nil {
				fmt.Printf("Got Error: %+v\n", evalErr.Error())
				os.Exit(1)
			}
			outPath := filepath.Join(viper.GetString("directory.output"), filePath)
			os.MkdirAll(filepath.Dir(outPath), os.ModePerm)
			fd, fileOpenErr := os.Create(outPath)
			if fileOpenErr != nil {
				fmt.Printf("Got Error: %+v\n", fileOpenErr.Error())
				os.Exit(1)
			}
			defer fd.Close()
			_, fileWriteErr := fd.Write(tplOut)
			if fileWriteErr != nil {
				fmt.Printf("Got Error: %+v\n", fileWriteErr.Error())
				os.Exit(1)
			}
		}

		copyErr := copy.Copy(viper.GetString("directory.static"), filepath.Join(viper.GetString("directory.output"), viper.GetString("directory.static")))
		if copyErr != nil {
			fmt.Printf("Got Error: %+v\n", copyErr.Error())
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

type fileList []string

func (s fileList) Len() int {
	return len(s)
}
func (s fileList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s fileList) Less(i, j int) bool {
	less := len(s[i]) < len(s[j])
	if less {
		return less
	} else {
		return strings.Compare(s[i], s[j]) < 0
	}
}
