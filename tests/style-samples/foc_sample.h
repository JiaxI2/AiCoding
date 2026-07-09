#ifndef FOC_SAMPLE_H
#define FOC_SAMPLE_H

typedef struct
{
    float id_ref;
    float iq_ref;
} FocController;

int foc_current_loop_update(FocController *ctrl, float id_ref, float iq_ref);

#endif
