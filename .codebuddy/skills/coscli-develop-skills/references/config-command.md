# Skill：配置命令开发（config 子命令）

## 概述

`config` 命令用于管理 `~/.cos.yaml` 配置文件，包含 `add`、`set`、`delete`、`show` 等子命令。
本 Skill 描述如何为配置文件新增字段，以及如何开发新的 config 子命令。

---

## 配置文件结构对应的 Go 结构体

```go
// util/types.go

// Config 对应 yaml 根节点 cos:
type Config struct {
    Base    BaseCfg  `yaml:"base"`
    Buckets []Bucket `yaml:"buckets"`
}

// BaseCfg 对应 cos.base
type BaseCfg struct {
    SecretID                   string `yaml:"secretid"`
    SecretKey                  string `yaml:"secretkey"`
    SessionToken               string `yaml:"sessiontoken"`
    Protocol                   string `yaml:"protocol"`
    Mode                       string `yaml:"mode"`
    CvmRoleName                string `yaml:"cvmrolename"`
    CloseAutoSwitchHost        string `yaml:"closeautoswitchhost"`
    DisableEncryption          string `yaml:"disableencryption"`
    DisableAutoFetchBucketType string `yaml:"disableautofetchbuckettype"`
}

// Bucket 对应 cos.buckets[] 中的每一项
type Bucket struct {
    Name       string `yaml:"name"`
    Alias      string `yaml:"alias"`
    Region     string `yaml:"region"`
    Endpoint   string `yaml:"endpoint"`
    Ofs        bool   `yaml:"ofs"`
    Customized bool   `yaml:"customized"`
}
```

---

## 为 Bucket 新增字段的完整步骤

以新增 `customized` 字段为例：

### 1. 修改 util/types.go

```go
type Bucket struct {
    Name       string `yaml:"name"`
    Alias      string `yaml:"alias"`
    Region     string `yaml:"region"`
    Endpoint   string `yaml:"endpoint"`
    Ofs        bool   `yaml:"ofs"`
    Customized bool   `yaml:"customized"`  // 新增
}
```

### 2. 修改 cmd/config_add.go

```go
func init() {
    configCmd.AddCommand(configAddCmd)
    configAddCmd.Flags().StringP("bucket", "b", "", "Bucket name")
    configAddCmd.Flags().StringP("endpoint", "e", "", "Bucket endpoint")
    configAddCmd.Flags().StringP("region", "r", "", "Bucket region")
    configAddCmd.Flags().StringP("alias", "a", "", "Bucket alias")
    configAddCmd.Flags().BoolP("ofs", "o", false, "Bucket ofs")
    configAddCmd.Flags().Bool("customized", false, "Use customized endpoint for this bucket")  // 新增
}

func addBucketConfig(cmd *cobra.Command) error {
    name, _ := cmd.Flags().GetString("bucket")
    // ...
    customized, _ := cmd.Flags().GetBool("customized")  // 新增

    bucket := util.Bucket{
        Name:       name,
        Endpoint:   endpoint,
        Region:     region,
        Alias:      alias,
        Ofs:        ofs,
        Customized: customized,  // 新增
    }
    // ...
    logger.Infof("Add successfully! name: %s, ..., customized: %t\n", name, ..., customized)
}
```

### 3. 在业务逻辑中使用（util/url.go）

```go
func GenURL(config *Config, param *Param, bucketName string) (*cos.BaseURL, error) {
    bucket, _, err := FindBucket(config, bucketName)
    // ...
    // 优先使用命令行参数，回退到配置文件
    customized := param.Customized
    if !customized {
        customized = bucket.Customized
    }
    return CreateURL(idName, protocol, endpoint, customized), nil
}
```

---

## config set 命令开发规范

`config set` 只修改 `base` 配置，使用 `@` 作为清空标记：

```go
func setConfigItem(cmd *cobra.Command) error {
    flag := false  // 标记是否有任何字段被修改

    someField, _ := cmd.Flags().GetString("some_field")
    if someField != "" {
        flag = true
        if someField == "@" {
            config.Base.SomeField = ""  // @ 表示清空
        } else {
            config.Base.SomeField = someField
        }
    }

    if !flag {
        return fmt.Errorf("Enter at least one configuration item to be modified!")
    }

    // 写回配置文件（标准模式）
    viper.Set("cos.base", config.Base)
    // ... 写文件逻辑
    logger.Infoln("Modify successfully!")
    return nil
}
```

---

## 密钥加密处理

`config set` 修改密钥时，若未禁用加密，需先加密再写入：

```go
// 写入前加密
if config.Base.DisableEncryption != "true" {
    config.Base.SecretKey, _ = util.EncryptSecret(config.Base.SecretKey)
    config.Base.SecretID, _ = util.EncryptSecret(config.Base.SecretID)
    config.Base.SessionToken, _ = util.EncryptSecret(config.Base.SessionToken)
}
```

`initConfig()` 读取时自动解密，命令层无需关心。

---

## 配置文件写入的 viper 模式

```go
// 修改 buckets 列表
config.Buckets = append(config.Buckets, bucket)
viper.Set("cos.buckets", config.Buckets)

// 修改 base 配置
viper.Set("cos.base", config.Base)

// 写入文件（统一模式，见 02-config-and-params.md）
```

---

## 桶别名规则

```go
// config add 中的别名校验
if alias == "" {
    alias = name  // 未指定别名时，别名等于桶名
}
for _, b := range config.Buckets {
    if name == b.Name {
        return fmt.Errorf("The bucket already exists, fail to add!")
    } else if alias == b.Name {
        return fmt.Errorf("The alias cannot be the same as other bucket's name")
    } else if alias == b.Alias {
        return fmt.Errorf("The alias already exists, fail to add!")
    }
}
```
