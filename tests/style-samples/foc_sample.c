#include "foc_sample.h"

int foc_current_loop_update(FocController *ctrl, float id_ref, float iq_ref)
{
    if (ctrl == 0)
    {
        return -1;
    }

    ctrl->id_ref = id_ref;
    ctrl->iq_ref = iq_ref;
    return 0;
}
