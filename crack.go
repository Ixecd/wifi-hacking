package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func expandHome(p string) string {
    if p == "" {
        return p
    }
    if strings.HasPrefix(p, "~") {
        home, _ := os.UserHomeDir()
        if home == "" {
            return p
        }
        return filepath.Join(home, strings.TrimPrefix(p, "~"))
    }
    return p
}

type wl struct {
    idx  int
    path string
}

func wordlists(dir string) []wl {
    ds, err := os.ReadDir(dir)
    if err != nil {
        return nil
    }
    out := make([]wl, 0, len(ds))
    for _, de := range ds {
        name := de.Name()
        if strings.HasSuffix(name, ".txt") {
            base := strings.TrimSuffix(name, ".txt")
            n, err := strconv.Atoi(base)
            if err == nil {
                out = append(out, wl{idx: n, path: filepath.Join(dir, name)})
            }
        }
    }
    sort.Slice(out, func(i, j int) bool { return out[i].idx < out[j].idx })
    return out
}

func run(wordlist string, pcap string, index int) (bool, error) {
    bin, err := exec.LookPath("aircrack-ng")
    if err != nil {
        fmt.Println("aircrack-ng未安装或不可用")
        return false, err
    }
    cmd := exec.Command(bin, "-w", wordlist, pcap)
    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()
    if err := cmd.Start(); err != nil {
        return false, err
    }
    found := false
    var sentIndex int32 = 0
	// 超时未收到索引提示时，自动输入索引
    time.AfterFunc(2*time.Second, func() {
        if atomic.CompareAndSwapInt32(&sentIndex, 0, 1) {
            fmt.Fprintf(stdin, "%d\n", index)
            stdin.Close()
        }
    })
    done := make(chan struct{}, 2)
    go func() {
        sc := bufio.NewScanner(stdout)
        for sc.Scan() {
            line := sc.Text()
            fmt.Println(line)
            l := strings.ToLower(line)
            if strings.Contains(l, "key found") {
                f, err := os.OpenFile("key.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
                if err != nil {
                    return
                }
                defer f.Close()
                f.WriteString(line + "\n")
				found = true
            }
            if strings.Contains(l, "index number of target network") {
                if atomic.CompareAndSwapInt32(&sentIndex, 0, 1) {
                    fmt.Fprintf(stdin, "%d\n", index)
                    stdin.Close()
                }
            }
        }
        done <- struct{}{}
    }()
    go func() {
        sc := bufio.NewScanner(stderr)
        for sc.Scan() {
            line := sc.Text()
            fmt.Println(line)
            l := strings.ToLower(line)
            if strings.Contains(l, "key found") {
                found = true
            }
            if strings.Contains(l, "index number of target network") {
                if atomic.CompareAndSwapInt32(&sentIndex, 0, 1) {
                    fmt.Fprintf(stdin, "%d\n", index)
                    stdin.Close()
                }
            }
        }
        done <- struct{}{}
    }()
    <-done
    <-done
    if err := cmd.Wait(); err != nil {
        return found, err
    }
    return found, nil
}

func main() {
    dirFlag := flag.String("dir", "data", "")
    pcapFlag := flag.String("pcap", "./pcap/qc_ch44_2025-11-19_19.39.38.424.pcap", "")
    indexFlag := flag.Int("index", 499, "")
    flag.Parse()
    dir := *dirFlag
    pcap := expandHome(*pcapFlag)
    lists := wordlists(dir)
    if len(lists) == 0 {
        fmt.Println("未找到字典文件")
        return
    }
    for _, w := range lists {
        fmt.Println("使用字典:", w.path)
		time.Sleep(1 * time.Second)
        ok, err := run(w.path, pcap, *indexFlag)
        if err != nil {
            fmt.Println("执行失败:", err)
        }
        if ok {
            fmt.Println("已找到密钥")
            return
        }
    }
    fmt.Println("未找到密钥")
}