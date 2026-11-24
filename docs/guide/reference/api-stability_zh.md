# API 边界与迁移

当前分支完整调整包路径与能力组合方式。旧目录及 Builder 能力开关不作为兼容 API 保留；当前 global 包继续作为兼容入口，转发到根注册表。升级清单见[迁移指南](../../design/module-migration_zh.md)。

公开区域包括根实例/默认门面、AuthType 选择器与共享注册表、global、contract、config、errors、event、独立能力包、组件包、框架集成与 SSO。internal 中的实现不供应用导入；需要命名的引擎扩展类型从根包导入。

已发布版本的使用方应核对目标根模块与扩展模块版本。框架便捷导出以 Gin 为标准源统一维护，应用首选根 Builder 和显式实例注入。发布前执行独立模块与仓库外消费者检查。

根包通过别名暴露的 Manager、Strategy 和装配类型仍属于公开兼容性范围；实现位于 internal 不取消其兼容性要求。根包包级业务方法与 Auth 方法签名一致；旧 global 函数的名称、选项和参数可能不同，参见 [Auth API](../../api/auth_zh.md)与[全局 API](../../api/dtoken_zh.md)。
