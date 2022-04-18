package file_dir_info

import (
	"fmt"
)

func OutputData(res []FileInfo)  {
	for _, f := range res {
		fmt.Printf("\tName: %s\t\t Path: %s\n", f.Name(), f.Path())
	}
}
