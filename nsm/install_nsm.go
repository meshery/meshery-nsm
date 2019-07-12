package nsm

import (
  "fmt"
   git "gopkg.in/src-d/go-git.v4"
   . "gopkg.in/src-d/go-git.v4/_examples"
)

func (nsmClient *NSMClient) download_NSM() {
        fmt.Println("downloading the project NSM ")
//        Info("git clone %s %s --recursive", "https://github.com/networkservicemesh/networkservicemesh.git","/home/harshini/Documents/networkservicemesh")

        r, err := git.PlainClone("/home/vagrant", false, &git.CloneOptions{
                URL:               "https://github.com/networkservicemesh/networkservicemesh.git" ,
                RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
        })

        CheckIfError(err)
        ref, err := r.Head()
        CheckIfError(err)
        commit, err := r.CommitObject(ref.Hash())
        CheckIfError(err)

        fmt.Println(commit)

}
