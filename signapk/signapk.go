package main

import (
	"log"
	"os"
	"path/filepath"
	"encoding/json"
	"io/ioutil"
	"github.com/jacpy/GoSamples/signapk/model"
	"fmt"
	"os/exec"
	"errors"
	"strings"
	"bufio"
	"io"
)

const kCMD_APK_BUILD = "-jar %s b %s -o %s"
const kCMD_APK_SIGN = "-verbose -keystore %s -storepass %s -signedjar %s %s %s"

var configInfo *model.ConfigInfo

func init()  {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	dir, _ := os.Getwd()
	// ./signapk/env.conf
	path := filepath.Join(dir, "signapk", "env.conf")
	_, err := os.Open(path)
	if os.IsNotExist(err) {
		log.Fatalln("file:", path, "is not exists.")
		return
	}

	err = parseConfig(path)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("java home:", configInfo.JavaHome, ", apktool:", configInfo.ApkTool,
		", key store:", configInfo.KeyStore, ", store password: ", configInfo.StorePwd)
}

/**
{
    "java_home": "",
    "apk_tool": "",
    "key_store": "",
    "store_password": "",
    "store_alias": ""
}
 */
func parseConfig(path string) error {
	f, _ := os.Open(path)
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	info := &model.ConfigInfo{}
	err = json.Unmarshal(b, info)
	if err != nil {
		return err
	}

	_, err = os.Open(info.JavaHome)
	if os.IsNotExist(err) {
		info.JavaHome = os.Getenv("JAVA_HOME")
	}

	err = checkConfig(info)
	if err != nil {
		log.Fatalln(err)
	}

	configInfo = info
	return nil
}

func checkConfig(config *model.ConfigInfo) error {
	binPath := filepath.Join(config.JavaHome, "bin")
	hasJava := false
	hasJarsigner := false
	filepath.Walk(binPath, func(path string, info os.FileInfo, err error) error {
		base := filepath.Base(path)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		if "java" == name {
			hasJava = true
		} else if "jarsigner" == name {
			hasJarsigner = true
		}

		return nil
	})

	if !hasJava {
		return errors.New(binPath + " does not has java executable file")
	}

	if !hasJarsigner {
		return errors.New(binPath + " does not has jarsinger executable file")
	}

	_, err := os.Open(config.ApkTool)
	if err != nil || os.IsNotExist(err) {
		return errors.New(config.ApkTool + " is not exist or not has read permission")
	}

	_, err = os.Open(config.KeyStore)
	if err != nil || os.IsNotExist(err) {
		return errors.New(config.KeyStore + " is not exist or not has read permission")
	}

	if len(config.StorePwd) <= 0 {
		return errors.New("store password is empty")
	}

	if len(config.StoreAlias) <= 0 {
		return errors.New("store alias is empty")
	}

	return nil
}

func pack(src string) (string, error) {
	file, err := os.Open(src)
	if os.IsNotExist(err) {
		return "", err
	}

	state, err := file.Stat()
	if err != nil {
		return "", err
	}

	if !state.IsDir() {
		return "", errors.New("input path is not directory")
	}

	dst := src + "-unsiged.apk"
	err = packApk(src, dst)
	if err != nil {
		return "", err
	}

	signedApk := src + "-signed.apk"
	err = signApk(dst, signedApk)
	if err != nil {
		return "", err
	}

	return signedApk, nil
}

func packApk(src, dst string) error {
	// java -jar apktool b src -o dst
	java := filepath.Join(configInfo.JavaHome, "bin", "java")
	cmdStr := fmt.Sprintf(kCMD_APK_BUILD, configInfo.ApkTool, src, dst)
	//cmd := exec.Command(java, "-jar", configInfo.ApkTool, "b", src, "-o", dst)
	cmd := exec.Command(java, strings.Split(cmdStr, " ")...)
	return printCmdOutput(cmd)
}

func signApk(src, dst string) error {
	// jarsigner -verbose -keystore keystore -signedjar signed.apk src.apk alias
	jarsigner := filepath.Join(configInfo.JavaHome, "bin", "jarsigner")
	cmdStr := fmt.Sprintf(kCMD_APK_SIGN, configInfo.KeyStore, configInfo.StorePwd, dst, src, configInfo.StoreAlias)
	cmd := exec.Command(jarsigner, strings.Split(cmdStr, " ")...)
	return printCmdOutput(cmd)
}

func printCmdOutput(cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	defer stdout.Close()
	defer stderr.Close()
	if err = cmd.Start(); err != nil {
		return err
	}

	br := bufio.NewReader(stdout)
	checkErrPipe := false
	isErrPipe := false
	for {
		buf, _, err := br.ReadLine()
		if err != nil || err == io.EOF {
			if !checkErrPipe && !isErrPipe {
				br = bufio.NewReader(stderr)
				isErrPipe = true
				continue
			}

			break
		}

		str := string(buf)
		if strings.Contains(str, " error:") {
			// apktool error log
			checkErrPipe = true
		}

		log.Println(str)
	}

	cmd.Wait()
	if checkErrPipe {
		return errors.New("execute error occur, please check output above")
	}

	return nil
}

func main() {
	log.Println("start sign apk...")
	if len(os.Args) < 2 {
		log.Fatalln("you must specify the unpack apk directory.")
	}

	src := os.Args[1]
	path, err := pack(src)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("sign apk success, path: ")
	log.Println(path)
}

