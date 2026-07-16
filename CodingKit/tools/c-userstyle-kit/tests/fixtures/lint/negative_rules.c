/**
 * @file negative_rules.c
 * @brief 仅供 lint 负例测试的故意违规输入。
 * @copyright Copyright (c) 2026 C UserStyle Kit.
 * @date 2026-07-15
 * @author C UserStyle Kit
 *
 * @details
 * 文件内容：集中放置机器可稳定识别的违规代码。
 * 主要功能：证明 lint 门禁会拒绝危险调用、变长数组、无界循环和不合规注释。
 * 文件关系：只由 Go 单元测试读取，禁止加入任何编译目标。
 */

uint32_t bad_global = 0U;

/**
 * @brief 这段注释故意错误地放在静态前置声明上。
 *
 * @return 无。
 */
static void DEMO_BadHelper(void);

/**
 * @brief 故意触发多项 lint 规则。
 *
 * @details 本函数不是可执行示例，只用于验证诊断 ID。
 *
 * @param[in] count 故意作为变长数组长度。
 *
 * @return 0；本返回值没有业务含义。
 */
int32_t DEMO_BadFixture(int32_t count)
{
    uint8_t scratch[count];
    void *buffer = malloc(8U); // 故意使用禁止的行注释和动态分配。

    switch (count)
    {
        case 0:
            break;

        default:
            break;
    }

    while (true)
    {
        count++;
    }

    return (buffer != NULL) ? (int32_t)scratch[0] : 0;
}
