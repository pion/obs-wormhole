package main

import (
	"fmt"
	"image/color"

	"fyne.io/fyne"
	"fyne.io/fyne/theme"
	"fyne.io/fyne/widget"
)

type Navigator interface {
	Push(path string, ctx interface{}) error
	Pop() error
}

type Page interface {
	fyne.Widget
	BeforeDestroy()
}

type routerRenderer struct {
	router *Router
}

func (r *routerRenderer) Layout(size fyne.Size) {
	r.router.currentPage.Resize(size)
}

func (r *routerRenderer) MinSize() fyne.Size {
	return r.router.currentPage.MinSize()
}

func (r *routerRenderer) BackgroundColor() color.Color {
	return theme.BackgroundColor()
}

func (r *routerRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.router.currentPage}
}

func (r *routerRenderer) Destroy() {}

func (r *routerRenderer) Refresh() {
	r.router.currentPage.Refresh()
}

type PageBuilder func(navigator Navigator, ctx interface{}) (Page, error)

type RouterConfig struct {
	routes      map[string]PageBuilder
	initialPath string
}

func (cfg *RouterConfig) Route(path string, pageBuilder PageBuilder) *RouterConfig {
	if cfg.routes == nil {
		cfg.routes = make(map[string]PageBuilder)
	}
	cfg.routes[path] = pageBuilder

	// by default if initialPath is not defined, used whatever path that was initialized first
	if cfg.initialPath == "" {
		cfg.initialPath = path
	}
	return cfg
}

func (cfg *RouterConfig) InitialPath(path string) *RouterConfig {
	cfg.initialPath = path
	return cfg
}

func (cfg *RouterConfig) Build() (*Router, error) {
	if len(cfg.routes) == 0 {
		return nil, fmt.Errorf("routes can't be empty")
	}

	buildInitialPage, ok := cfg.routes[cfg.initialPath]
	if !ok {
		return nil, fmt.Errorf("%s doesn't exist in routes: %v", cfg.initialPath, cfg.routes)
	}

	router := Router{
		RouterConfig: cfg,
	}
	initialPage, err := buildInitialPage(&router, nil)
	if err != nil {
		return nil, err
	}
	router.currentPage = initialPage
	router.ExtendBaseWidget(&router)
	return &router, nil
}

type Router struct {
	widget.BaseWidget
	*RouterConfig
	history     []Page
	currentPage Page
}

func (router *Router) CreateRenderer() fyne.WidgetRenderer {
	return &routerRenderer{router: router}
}

func (router *Router) Push(path string, ctx interface{}) error {
	buildNextPage, ok := router.routes[path]
	if !ok {
		return fmt.Errorf("%s doesn't exist in routes: %v", path, router.routes)
	}

	router.history = append(router.history, router.currentPage)
	nextPage, err := buildNextPage(router, ctx)
	if err != nil {
		return err
	}
	router.currentPage = nextPage
	router.Refresh()
	return nil
}

func (router *Router) Pop() error {
	if len(router.history) == 0 {
		return fmt.Errorf("there's no more pages in history")
	}

	router.currentPage.BeforeDestroy()
	router.currentPage = router.history[len(router.history)-1]
	router.history = router.history[:len(router.history)-1]
	router.Refresh()
	return nil
}
