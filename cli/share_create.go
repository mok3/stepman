package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/pointers"
	"github.com/bitrise-io/goinp/goinp"
	"github.com/bitrise-io/stepman/models"
	"github.com/bitrise-io/stepman/stepman"
	"github.com/codegangsta/cli"
)

func printFinishCreate(share ShareModel, stepDirInSteplib string, toolMode bool) {
	fmt.Println()
	log.Infof(" * "+colorstring.Green("[OK]")+" Your step (%s) (%s) added to local StepLib (%s).", share.StepID, share.StepTag, stepDirInSteplib)
	log.Infoln(" *      You can find your Step's step.yml at: " + colorstring.Greenf("%s/step.yml", stepDirInSteplib))
	fmt.Println()
	fmt.Println("   " + GuideTextForShareFinish(toolMode))
}

func getStepIDFromGit(git string) string {
	splits := strings.Split(git, "/")
	lastPart := splits[len(splits)-1]
	splits = strings.Split(lastPart, ".")
	return splits[0]
}

func create(c *cli.Context) {
	toolMode := c.Bool(ToolMode)

	share, err := ReadShareSteplibFromFile()
	if err != nil {
		log.Error(err)
		log.Fatal("You have to start sharing with `stepman share start`, or you can read instructions with `stepman share`")
	}

	// Input validation
	tag := c.String(TagKey)
	if tag == "" {
		log.Fatalln("[STEPMAN] - No step tag specified")
	}

	gitURI := c.String(GitKey)
	if gitURI == "" {
		log.Fatalln("[STEPMAN] - No step url specified")
	}

	stepID := c.String(StepIDKEy)
	if stepID == "" {
		stepID = getStepIDFromGit(gitURI)
	}
	if stepID == "" {
		log.Fatalln("No step id specified")
	}

	route, found := stepman.ReadRoute(share.Collection)
	if !found {
		log.Fatalln("No route found for collectionURI (%s)", share.Collection)
	}
	stepDirInSteplib := stepman.GetStepCollectionDirPath(route, stepID, tag)
	stepYMLPathInSteplib := stepDirInSteplib + "/step.yml"
	if exist, err := pathutil.IsPathExists(stepYMLPathInSteplib); err != nil {
		log.Fatal(err)
	} else if exist {
		log.Warnf("[STEPMAN] - step.yml already exist in path: %s.", stepDirInSteplib)
		if val, err := goinp.AskForBool("Would you like to overwrite local version of step.yml? [yes/no]"); err != nil {
			log.Fatalln("Error:", err)
		} else {
			if !val {
				log.Errorln("Unfortunately we can't continue with sharing without an overwrite exist step.yml.")
				log.Fatalln("Please finish your changes, run this command again and allow it to overwrite the exist step.yml!")
				return
			}
		}
	}

	// Clone step to tmp dir
	tmp, err := pathutil.NormalizedOSTempDirPath("")
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Cloning step from (%s) with tag (%s) to temporary path (%s)", gitURI, tag, tmp)
	if err := cmdex.GitCloneTagOrBranch(gitURI, tmp, tag); err != nil {
		log.Fatal(err)
	}

	// Update step.yml
	bytes, err := fileutil.ReadBytesFromFile(tmp + "/step.yml")
	if err != nil {
		log.Fatal(err)
	}
	var stepModel models.StepModel
	if err := yaml.Unmarshal(bytes, &stepModel); err != nil {
		log.Fatal(err)
	}
	commit, err := cmdex.GitGetCommitHashOfHEAD(tmp)
	if err != nil {
		log.Fatal(err)
	}
	stepModel.Source = models.StepSourceModel{
		Git:    gitURI,
		Commit: commit,
	}
	stepModel.PublishedAt = pointers.NewTimePtr(time.Now())
	if err := stepModel.Validate(true); err != nil {
		log.Fatal(err)
	}

	// Copy step.yml to steplib
	share.StepID = stepID
	share.StepTag = tag
	if err := WriteShareSteplibToFile(share); err != nil {
		log.Fatal("[STEPMAN] - Failed to save share steplib to file:", err)
	}

	log.Info("Step dir in collection:", stepDirInSteplib)
	if exist, err := pathutil.IsPathExists(stepDirInSteplib); err != nil {
		log.Fatal(err)
	} else if !exist {
		if err := os.MkdirAll(stepDirInSteplib, 0777); err != nil {
			log.Fatal(err)
		}
	}

	log.Info("Checkout branch:", share.StepID)
	collectionDir := stepman.GetCollectionBaseDirPath(route)
	if err := cmdex.GitCheckout(collectionDir, share.StepID); err != nil {
		if err := cmdex.GitCreateAndCheckoutBranch(collectionDir, share.StepID); err != nil {
			log.Fatal(err)
		}
	}

	stepBytes, err := yaml.Marshal(stepModel)
	if err != nil {
		log.Fatal(err)
	}
	if err := fileutil.WriteBytesToFile(stepYMLPathInSteplib, stepBytes); err != nil {
		log.Fatal(err)
	}
	// Update spec.json
	if err := stepman.ReGenerateStepSpec(route); err != nil {
		log.Fatal(err)
	}

	printFinishCreate(share, stepDirInSteplib, toolMode)
}
