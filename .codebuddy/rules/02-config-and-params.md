# 规则：参数优先级与配置文件

## 三层优先级规则

所有可配置参数遵循：**命令行参数 > 配置文件桶级别 > 配置文件 base 级别**

### String 类型参数

```go
// 以 endpoint 为例（见 util/url.go GenURL 函数）
endpoint := bucket.Endpoint          // 第三优先级：桶配置
if param.Endpoint != "" {
    endpoint = param.Endpoint        // 第一优先级：命令行参数
}
if endpoint == "" && bucket.Region != "" {
    endpoint = fmt.Sprintf("cos.%s.myqcloud.com", bucket.Region)  // 兜底：region 推导
}
```

### Bool 类型参数（customized 为标准示例）

```go
// 见 util/url.go GenURL 函数
customized := param.Customized       // 第一优先级：命令行参数（--customized）
if !customized {
    customized = bucket.Customized   // 第二优先级：桶配置文件
}
// 零值 false = 不启用，天然兼容历史配置文件（无该字段时反序列化为 false）
```

### String 开关类型（如 CloseAutoSwitchHost）

```go
// 见 util/client.go NewClient 函数
CloseAutoSwitchHost := param.CloseAutoSwitchHost
if CloseAutoSwitchHost == "" {
    CloseAutoSwitchHost = config.Base.CloseAutoSwitchHost
}
```

## 配置文件结构（~/.cos.yaml）

```yaml
cos:
  base:
    secretid: "xxx"
    secretkey: "xxx"
    sessiontoken: ""
    protocol: "https"
    mode: "SecretKey"          # SecretKey 或 CvmRole
    cvmrolename: ""
    closeautoswitchhost: ""    # "false" 表示开启备用域名切换
    disableencryption: ""      # "true" 表示禁用密钥加密
    disableautofetchbuckettype: ""  # "true" 表示禁用自动获取桶类型
  buckets:
    - name: "examplebucket-1234567890"
      alias: "example"
      region: "ap-guangzhou"
      endpoint: "cos.ap-guangzhou.myqcloud.com"
      ofs: false
      customized: false        # true 表示使用自定义域名（endpoint 作为完整域名）
```

## 配置文件读取（自动，无需手动）

`initConfig()` 在每次命令执行前自动运行，将配置反序列化到全局 `config` 变量：

```go
// root.go initConfig() 中
if err := viper.UnmarshalKey("cos", &config); err != nil {
    fmt.Println(err)
    os.Exit(1)
}
```

命令的 `RunE` 中直接使用 `config` 和 `param`，无需手动读取配置文件。

## 配置文件写入规范

所有写入配置文件的操作必须遵循以下模式（见 `cmd/config_add.go`）：

```go
home, err := homedir.Dir()
configFile := ""
if cfgFile != "" {
    if cfgFile[0] == '~' {
        configFile = home + cfgFile[1:]
    } else {
        configFile = cfgFile
    }
} else {
    configFile = home + "/.cos.yaml"
}
_, err = os.Stat(configFile)
if os.IsNotExist(err) || cfgFile != "" {
    if err := viper.WriteConfigAs(configFile); err != nil {
        return err
    }
} else {
    if err := viper.WriteConfigAs(viper.ConfigFileUsed()); err != nil {
        return err
    }
}
```

## 新增配置字段必须同步修改四处

新增一个配置字段时，必须同步修改：

1. **`util/types.go`** — 在对应结构体中添加字段（含 `yaml` tag）
2. **`cmd/config_add.go`** — 添加 Flag 声明和写入逻辑
3. **`cmd/config_set.go`** — 添加 Flag 声明和写入逻辑（base 字段）
4. **`util/url.go` 或对应业务文件** — 读取并应用该字段

## 布尔字段兼容历史配置文件

布尔字段的零值必须为安全默认值（`false` = 关闭特性），确保历史配置文件（无该字段）反序列化后行为不变：

```go
// ✅ 正确：零值 false = 不启用自定义域名（安全默认）
Customized bool `yaml:"customized"`

// ✅ 正确：零值 false = 不是 OFS 桶（安全默认）
Ofs bool `yaml:"ofs"`

// ❌ 错误：若零值需要特殊处理，必须用指针或 string 类型
```
