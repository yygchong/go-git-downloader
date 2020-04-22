package main

import ("fmt"
"io/ioutil"
"encoding/xml"
"os/exec"
"os"
flag "github.com/spf13/pflag"
"sync"
"strconv"
PQ "github.com/bgadrian/data-structures/priorityqueue"
)

type Manifest struct{
  ProjectList []Project `xml:"Project"`
}

type Project struct {
  Filename string `xml:"filename"`
  Url string `xml:"url"`
  Branch string `xml:"branch"`
  Commit string `xml:"commit"`
}

func exportResult(wg *sync.WaitGroup, p Project) {
  defer wg.Done()
  //Converting Github SSH Url to HTTPS to use with local git method
  var endOfUrl = p.Url[15:]
  var startOfUrl = "https://github.com/"
  //Finding and segmenting project name from folder name
  var folderName = ""
  var projectName = ""
  for j:=0 ; j < len(p.Filename); j++ {
    if p.Filename[j] == '/' {
      folderName = p.Filename[0:j]
      projectName = p.Filename[j+1:]
      break
    }
  }
  var newUrl = startOfUrl + endOfUrl
  //Executing local git clone command using branch
  if(p.Branch == ""){
    p.Branch = "master"
  }
  cmd := exec.Command("git", "clone", "-b", p.Branch, newUrl)
  _, err := cmd.Output()
  if err != nil {
      fmt.Println(err.Error())
      return
  }

  //If a specific commit SHA is required then we execute "git reset --hard SHA#"
  if p.Commit != "" {
    //Change directory into git directory in order to completion operation
    os.Chdir(projectName)
    cmd2 := exec.Command("git", "reset", "--hard", p.Commit)
    _, err2 := cmd2.Output()
    if err != nil {
        fmt.Println(err2.Error())
        return
    }
    //Return to outer directory
    os.Chdir("..")
  }
  fmt.Println("Successfully cloned", projectName, "into", folderName)
}

func main() {
  //Creating cmd line flags from pflag
  var inputFlag = flag.String("input", "", "input help message" )
  var outputFlag = flag.String("outputFolder", "", "output help message" )
  flag.Parse()
  bytes, _ := ioutil.ReadFile(*inputFlag)
  //Create a new directory with the name specified by the outputFlag
  os.MkdirAll(*outputFlag, os.ModePerm)
  os.Chdir(*outputFlag)
  var manifest Manifest
  //Parse XML file into manifest variable
  xml.Unmarshal(bytes, &manifest)
  //Create a map where the keys are folders (ex: folder2/..., folder2/...)
  //and where the values are PriorityQueues which contain the priorities of the projects
  folderMap := make(map[string]*PQ.HierarchicalQueue)

  //For loop that iterates through the ProjectList indices
  for i := range manifest.ProjectList {
    //Finds the name of the folder in the current ProjectList index
    var folderName = ""
    for j:=0 ; j < len(manifest.ProjectList[i].Filename); j++ {
      if manifest.ProjectList[i].Filename[j] == '/' {
        folderName = manifest.ProjectList[i].Filename[0:j]
        break
      }
    }
    //Checks our map to see whether or not the current folder name is already in our map
    //(ex: folderMap[folder1])
    if(folderMap[folderName] == nil){
      //If the folder name does not exisit in our map we create a new priority queue
      pq :=  PQ.NewHierarchicalQueue(uint8(len(manifest.ProjectList)), false)
      folderMap[folderName] = pq
    }

    folderMap[folderName].Enqueue(strconv.Itoa(i), 1)
  }
  //For loop to iterate through each key in the map
  for k := range folderMap {
    var wg sync.WaitGroup
    //While loop that keeps popping values from folder until it is empty
    for folderMap[k].Len() != 0 {
      wg.Add(1)
      var newNode, _ = folderMap[k].Dequeue()

      var i, _ = strconv.Atoi(newNode.(string))
      //Calls helper function in the order that projects are added into the priority queue
      go exportResult(&wg, manifest.ProjectList[i])
    }
      wg.Wait()
  }
}
