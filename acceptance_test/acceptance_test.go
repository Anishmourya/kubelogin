package acceptance_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/sync/errgroup"
)

const kubeconfigEnv = "KUBECONFIG=testdata/kubeconfig.yaml:kubeconfig.yaml"

func init() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
}

func Test(t *testing.T) {
	if err := os.RemoveAll("testdata/token-cache"); err != nil {
		t.Fatalf("could not remove the token cache: %s", err)
	}
	ctx := context.TODO()
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error { return runKubectl(ctx, t, eg) })
	eg.Go(func() error { return openBrowserAndLogInToDex(ctx, t) })
	if err := eg.Wait(); err != nil {
		t.Errorf("error: %s", err)
	}
}

func runKubectl(ctx context.Context, t *testing.T, eg *errgroup.Group) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.Command("kubectl", "--user=oidc", "-n", "dex", "get", "deploy")
	cmd.Env = append(os.Environ(), kubeconfigEnv)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	eg.Go(func() error {
		<-ctx.Done()
		if cmd.Process == nil {
			log.Printf("process not started")
			return nil
		}
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			log.Printf("process terminated with exit code %d", cmd.ProcessState.ExitCode())
			return nil
		}
		log.Printf("sending SIGTERM to pid %d", cmd.Process.Pid)
		// kill the child processes
		// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
			t.Errorf("could not send a signal: %s", err)
		}
		return nil
	})
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not run a command: %w", err)
	}
	return nil
}

func openBrowserAndLogInToDex(ctx context.Context, t *testing.T) error {
	time.Sleep(10 * time.Second)

	ctx, cancel1 := chromedp.NewContext(ctx, chromedp.WithDebugf(log.Printf))
	defer cancel1()
	ctx, cancel2 := context.WithTimeout(ctx, 20*time.Second)
	defer cancel2()
	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(`http://localhost:8000`),
		// https://server.dex.svc.cluster.local:30443/dex/auth/local
		chromedp.WaitVisible(`#login`),
		printLocation(),
		chromedp.SendKeys(`#login`, `admin@example.com`),
		chromedp.SendKeys(`#password`, `password`),
		chromedp.Submit(`#submit-login`),
		// https://server.dex.svc.cluster.local:30443/dex/approval
		chromedp.WaitVisible(`.dex-btn.theme-btn--success`),
		printLocation(),
		chromedp.Submit(`.dex-btn.theme-btn--success`),
		// http://localhost:8000
		chromedp.WaitReady(`body`),
		printLocation(),
		chromedp.Text(`body`, &body),
	)
	if err != nil {
		return fmt.Errorf("could not run the browser test: %w", err)
	}
	if body != "OK" {
		t.Errorf("body wants OK but got %s", body)
	}
	return nil
}

func printLocation() chromedp.Action {
	var location string
	return chromedp.Tasks{
		chromedp.Location(&location),
		chromedp.ActionFunc(func(ctx context.Context) error {
			log.Printf("location: %s", location)
			return nil
		}),
	}
}
