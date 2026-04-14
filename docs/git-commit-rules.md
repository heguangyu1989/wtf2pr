
## git提交规范
-----

### 一、目标
- 规范化提交，表述清晰每次提交修改内容
- 依据提交可以自动审查，自动生成change log


### 二、参考资料
- [Commit message 和 Change log 编写指南](https://www.ruanyifeng.com/blog/2016/01/commit_message_change_log.html)

### 三、规则
1. 每次提交都要写明提交信息，否则不允许提交
2. 每次提交信息都需要符合[提交格式](#3.1)，否则不允许提交
3. 每次提交都处理一个问题，不要多个问题或者修复混合在一起
4. 提交语言除了规定参数取值外，尽量都已中文信息作为提交信息
5. 参数描述中不要存在空格，因为空格将要作为分隔符


### 3.1 提交格式与参数说明
**提交格式**
> 以{参数名称: 必要程度}这样的格式表示提交参数。其中必要程度: must, should, optional


```bash
{type: must} {breaking: optional} {subject: must} {jiralink: should}
// 空一行
{body: optional} 
```

#### 3.1.1 type
> type限制在如下列表选择

- feat: 新功能
- fix: 修补bug
- docs: 文档（documentation）
- style: 格式（不影响代码运行的变动）
- refactor: 重构（即不是新增功能，也不是修改bug的代码变动）
- test: 增加测试
- chore: 构建过程或辅助工具的变动
- revert: 回滚
- enhance: 能力增强

#### 3.1.2 breaking
当参数不填写的时候默认本次提交与先前版本是兼容的。当写入breaking的时候表示盖茨提交造成不兼容。

#### 3.1.3 subject
主题，尽量用简短的文字说明本次提交的主题

#### 3.1.5 body
剪短必要的说明文字











