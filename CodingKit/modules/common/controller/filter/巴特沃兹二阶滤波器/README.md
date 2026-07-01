# motor_bw_lpf

纯 C 二阶 Butterworth 低通滤波器，面向嵌入式电机控制电流采样链路。

## 文件

- `motor_bw_lpf.h`
- `motor_bw_lpf.c`
- `tests/test_motor_bw_lpf.c`

## 最小用法

```c
#include "motor_bw_lpf.h"

static motor_bw_lpf2_t ia_lpf;
static motor_bw_lpf2_t ib_lpf;

void CurrentLoop_Init(void)
{
    (void)MotorBwLpf2_Init(&ia_lpf, 20000.0f, 2000.0f, 0.0f);
    (void)MotorBwLpf2_Init(&ib_lpf, 20000.0f, 2000.0f, 0.0f);
}

void CurrentLoop_Isr(void)
{
    float ia_raw = ADC_GetIa();
    float ib_raw = ADC_GetIb();
    float ia = MotorBwLpf2_Update(&ia_lpf, ia_raw);
    float ib = MotorBwLpf2_Update(&ib_lpf, ib_raw);

    /* ia/ib -> Clarke/Park -> PI */
}
```

## 电流环建议

用于闭环反馈时，截止频率不要过低。建议先按：

```text
fc_filter >= 5 ~ 10 * current_loop_bandwidth
```

例如 FOC/PWM ISR 为 20 kHz、电流环目标带宽 1 kHz，可以先测试 `fc=5 kHz`，噪声仍大再降到 `2~3 kHz`，并用阶跃或频响验证相位裕度。

## 主机测试

```bash
gcc -std=c99 -Wall -Wextra -pedantic \
  motor_bw_lpf.c tests/test_motor_bw_lpf.c -lm -o test_motor_bw_lpf
./test_motor_bw_lpf
```
