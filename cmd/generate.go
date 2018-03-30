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

		files := new(fileSet)
		gatherErr := filepath.Walk(viper.GetString("directory.content"), func(path string, f os.FileInfo, err error) error {
			if !f.IsDir() {
				files.addItem(path)
			}
			return nil
		})
		if gatherErr != nil {
			fmt.Printf("Got Error: %+v\n", gatherErr.Error())
			os.Exit(1)
		}

		files.sortFiles()

		fs := pongo2.MustNewLocalFileSystemLoader("")
		s = pongo2.NewSet("test set with base directory", fs)
		s.Globals["base_directory"] = pwd
		if initErr := fs.SetBaseDir(s.Globals["base_directory"].(string)); initErr != nil {
			fmt.Printf("Got Error: %+v\n", initErr.Error())
			os.Exit(1)
		}

		for _, file := range files.Files {
			fmt.Printf("Visiting: %s\n", file.path)

			context := pongo2.Context{
				"dir":         file.dir,
				"path":        file.contentPath,
				"environment": viper.GetStringMap("environment"),
			}

			tpl, dtlErr := s.FromFile(file.path)
			if dtlErr != nil {
				fmt.Printf("Got Error: %+v\n", dtlErr.Error())
				os.Exit(1)
			}

			tplOut, evalErr := tpl.ExecuteBytes(context)
			if evalErr != nil {
				fmt.Printf("Got Error: %+v\n", evalErr.Error())
				os.Exit(1)
			}

			os.MkdirAll(filepath.Dir(file.outPath), os.ModePerm)

			fd, fileOpenErr := os.Create(file.outPath)
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

type parsedFile struct {
	path        string
	name        string
	contentPath string
	outPath     string
	dir         string
	parts       []string
}

type fileSet struct {
	Files []parsedFile
}

func (this *fileSet) addItem(path string) error {
	contentPath := strings.TrimPrefix(path, viper.GetString("directory.content"))
	outPath := filepath.Join(viper.GetString("directory.output"), contentPath)
	dir, name := filepath.Split(contentPath)
	parts := strings.Split(contentPath, string(filepath.Separator))
	this.Files = append(this.Files, parsedFile{
		path:        path,
		name:        name,
		contentPath: contentPath,
		outPath:     outPath,
		dir:         dir,
		parts:       parts,
	})
	return nil
}

func (this *fileSet) sortFiles() error {
	sort.Sort(this)
	return nil
}

func (s fileSet) Len() int {
	return len(s.Files)
}
func (s fileSet) Swap(i, j int) {
	s.Files[i], s.Files[j] = s.Files[j], s.Files[i]
}
func (s fileSet) Less(i, j int) bool {
	less := len(s.Files[i].parts) < len(s.Files[j].parts)
	if less {
		return less
	} else {
		return strings.Compare(s.Files[i].name, s.Files[j].name) < 0
	}
}
