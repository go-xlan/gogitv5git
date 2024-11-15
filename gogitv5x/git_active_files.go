package gogitv5x

import (
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/yyle88/erero"
	"github.com/yyle88/osexistpath/ossoftexist"
	"github.com/yyle88/zaplog"
	"go.uber.org/zap"
)

type GetActiveFilesOptions struct {
	Root                string // project location path. 目前还不知道怎么从他们给的结构里拿到，因此只能再次传进来，比较无奈呢
	IncludeDeletedFiles bool   // default false(results not include deleted files)
	FileExtension       string // ".go" / ".txt". match the file name extension
	NoneExtension       bool   // match path without extension.

	MatchPathFunc func(string) bool
	RunOnEachPath func(path string) error
}

func NewGetActiveFilesOptions(root string) *GetActiveFilesOptions {
	return &GetActiveFilesOptions{
		Root: root,
	}
}

func (T *GetActiveFilesOptions) SetIncludeDeletedFiles(include bool) *GetActiveFilesOptions {
	T.IncludeDeletedFiles = include
	return T
}

func (T *GetActiveFilesOptions) SetFileExtension(extension string) *GetActiveFilesOptions {
	T.FileExtension = extension
	return T
}

func (T *GetActiveFilesOptions) SetNoneExtension(none bool) *GetActiveFilesOptions {
	T.NoneExtension = none
	return T
}

func (T *GetActiveFilesOptions) SetMatchPathFunc(fn func(path string) bool) *GetActiveFilesOptions {
	T.MatchPathFunc = fn
	return T
}

func (T *GetActiveFilesOptions) SetRunOnEachPath(fn func(path string) error) *GetActiveFilesOptions {
	T.RunOnEachPath = fn
	return T
}

func (T *GetActiveFilesOptions) SetRunOnFilePath(fn func(path string)) *GetActiveFilesOptions {
	return T.SetRunOnEachPath(func(path string) error {
		zaplog.ZAPS.P1.LOG.Info("run-on-path", zap.String("path", path))
		fn(path)
		return nil
	})
}

// GetActiveFiles 找到变化的文件（除了删除的）把变动的文件格式化再提交
func GetActiveFiles(worktree *git.Worktree, option *GetActiveFilesOptions) (activeFiles []string, err error) {
	statusMap, err := worktree.Status()
	if err != nil {
		return nil, erero.Wro(err)
	}

	activeFiles = make([]string, 0, len(statusMap))

	for subPath, sts := range statusMap {
		// 需要过滤掉已经删除的，默认不包含已经删除的
		if !option.IncludeDeletedFiles && sts.Staging == git.Deleted {
			continue
		}

		//需要过滤掉扩展名不匹配的
		if option.FileExtension != "" {
			if ext := filepath.Ext(subPath); ext != option.FileExtension {
				continue
			}
		}

		//当需要匹配不含扩展名的时，就用这个逻辑
		if option.NoneExtension {
			if ext := filepath.Ext(subPath); ext != "" {
				continue
			}
		}

		var resPath string //收集文件路径
		if option.Root != "" {
			absPath := filepath.Join(option.Root, subPath)
			resPath = absPath

			//当需要过滤路径时，就可以通过这个函数过滤，把不需要处理的路径排除掉
			if option.MatchPathFunc != nil && !option.MatchPathFunc(absPath) {
				continue
			}

			//当需要对这个文件执行特殊操作时，把操作传进来起，操作可以是修改这个文件的内容
			if option.RunOnEachPath != nil {
				//这时就得要求这个文件是存在的，而不是被删除的，或者不存在的
				if sts.Staging != git.Deleted && ossoftexist.IsFile(absPath) {
					//这里的操作可以是打印文件内容
					//当然可以是修改文件内容-比如替换文件内容-比如格式化go代码
					if err := option.RunOnEachPath(absPath); err != nil {
						return nil, erero.Wro(err)
					}
				}
			}
		} else {
			resPath = subPath

			//排除掉一些没用的路径
			if option.MatchPathFunc != nil && !option.MatchPathFunc(subPath) {
				continue
			}
		}
		activeFiles = append(activeFiles, resPath) //收集文件路径
	}
	return activeFiles, nil
}
