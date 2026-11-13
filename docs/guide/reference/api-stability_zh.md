# API 边界与迁移

当前分支完整调整包路径与能力组合方式。旧目录、旧全局入口及 Builder 能力开关不作为兼容 API 保留。升级清单见[迁移指南](../../design/module-migration_zh.md)。

公开区域包括根实例门面、global、contract、config、errors、event、独立能力包、组件包、框架集成与 SSO。internal 中的实现不供应用导入；需要命名的引擎扩展类型从根包导入。

已发布版本的使用方应核对目标根模块与扩展模块版本。框架便捷导出以 Gin 为标准源统一维护，应用首选根 Builder 和显式实例注入。发布前执行独立模块与仓库外消费者检查。
