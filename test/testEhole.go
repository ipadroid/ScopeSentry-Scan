// main-------------------------------------
// @file      : testEhole.go
// @author    : Autumn
// @contact   : rainy-autumn@outlook.com
// @time      : 2024/11/30 17:24
// -------------------------------------------

package main

import (
	"encoding/json"
	"fmt"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/bigcache"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/config"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/configupdater"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/contextmanager"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/global"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/handler"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/mongodb"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/notification"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/pebbledb"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/plugins"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/pool"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/redis"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/results"
	"github.com/Autumn-27/ScopeSentry-Scan/internal/types"
	"github.com/Autumn-27/ScopeSentry-Scan/pkg/logger"
	"github.com/Autumn-27/ScopeSentry-Scan/pkg/utils"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// 初始化系统信息
	config.Initialize()
	var err error
	// 初始化mongodb连接
	mongodb.Initialize()
	// 初始化redis连接
	redis.Initialize()
	// 初始化日志模块
	err = logger.NewLogger()
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	// 初始化任务计数器
	handler.InitHandle()
	// 更新配置、加载字典
	configupdater.Initialize()
	// 初始化模块配置
	err = config.ModulesInitialize()
	if err != nil {
		log.Fatalf("Failed to init ModulesConfig: %v", err)
		return
	}
	// 初始化上下文管理器
	contextmanager.NewContextManager()
	// 初始化tools
	utils.InitializeTools()
	utils.InitializeDnsTools()
	utils.InitializeRequests()
	utils.InitializeResults()
	// 初始化通知模块
	notification.InitializeNotification()
	// 初始化协程池
	pool.Initialize()
	// 初始化个模块的协程池
	pool.PoolManage.InitializeModulesPools(config.ModulesConfig)
	go pool.StartMonitoring()
	// 初始化内存缓存
	err = bigcache.Initialize()
	if err != nil {
		logger.SlogErrorLocal(fmt.Sprintf("bigcache Initialize error: %v", err))
		return
	}
	// 初始化本地持久化缓存
	pebbledbSetting := pebbledb.Settings{
		DBPath:       filepath.Join(global.AbsolutePath, "data", "pebbledb"),
		CacheSize:    64 << 20,
		MaxOpenFiles: 500,
	}
	pebbledbOption := pebbledb.GetPebbleOptions(&pebbledbSetting)
	if !global.AppConfig.Debug {
		pebbledbOption.Logger = nil
	}
	pedb, err := pebbledb.NewPebbleDB(pebbledbOption, pebbledbSetting.DBPath)
	if err != nil {
		return
	}
	pebbledb.PebbleStore = pedb
	defer func(PebbleStore *pebbledb.PebbleDB) {
		_ = PebbleStore.Close()
	}(pebbledb.PebbleStore)

	// 初始化结果处理队列，(正常的时候将该初始化放入任务开始时，任务执行完毕关闭结果队列)
	results.InitializeResultQueue()
	defer results.Close()

	// 初始化全局插件管理器
	plugins.GlobalPluginManager = plugins.NewPluginManager()
	err = plugins.GlobalPluginManager.InitializePlugins()
	if err != nil {
		log.Fatalf("Failed to init plugins: %v", err)
		return
	}
	var assetsList []types.AssetHttp
	// 打开文件
	file, err := os.Open("ScopeSentry.asset.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// 读取文件内容
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	// 将 JSON 数据解析到结构体
	err = json.Unmarshal(byteValue, &assetsList)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(len(assetsList))
	plg, _ := plugins.GlobalPluginManager.GetPlugin("AssetHandle", "4fb56e0137e9fda3e9c76e9446868fbb")
	plg.SetParameter("-finger 674ac1d1f11d1fdbe5dd6f01 -thread 30")
	for _, as := range assetsList {
		//fmt.Printf("%v", as)
		plg.Execute(&as)
	}
}
