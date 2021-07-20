#include "hp.h"

HARP_child_t HARP_children[6];

int sortChildrenByIfaceTs(const void *a, const void *b)
{
    return ((HARP_child_t *)b)->iface[0].ts - ((HARP_child_t *)a)->iface[0].ts;
}

int sortChildrenByIfaceCh(const void *a, const void *b)
{
    return ((HARP_child_t *)b)->iface[0].ch - ((HARP_child_t *)a)->iface[0].ch;
}

void initChildren()
{
    uint8_t rects[6][3] = {{6, 1, 32}, {8, 2, 27}, {9, 1, 16}, {1, 1, 24}, {2, 2, 23}, {2, 1, 29}};
    for (int i = 0; i < 6; i++)
    {
        HARP_children[i].id = rects[i][2];
        HARP_children[i].iface[0].ts = rects[i][0];
        HARP_children[i].iface[0].ch = rects[i][1];
    }
    // qsort(HARP_children, 6, sizeof(HARP_child_t), sortChildrenByIfaceTs);
    // printf("first HARP_child_t's interface {ts:%d, ch:%d}\n", HARP_children[0].iface[0].ts, HARP_children[0].iface[0].ch);
    // printf("first HARP_child_t's sub-partition {%d, %d}\n", HARP_children[0].sp_[0].ts_start,  HARP_children[0].sp_log[0].ts_end);
}


uint8_t width, height;

uint8_t skylinePacking()
{
    int layer = 0;
    width = 0;
    height = 0;
    printf("Best-fit Skyline Packing\n");

    qsort(HARP_children, 6, sizeof(HARP_child_t), sortChildrenByIfaceTs);

    HARP_children[0].sp_log[layer].ts_start = 0;
    HARP_children[0].sp_log[layer].ts_end = HARP_children[0].iface[layer].ts;
    HARP_children[0].sp_log[layer].ch_start = 0;
    HARP_children[0].sp_log[layer].ch_end = 0 + HARP_children[layer].iface[layer].ch;

    width = HARP_children[0].iface[layer].ts;

    HARP_skyline_t *skyline = (HARP_skyline_t *)malloc(sizeof(HARP_skyline_t));
    skyline->start = 0;
    skyline->end = width;
    skyline->width = width;
    skyline->height = HARP_children[0].iface[layer].ch;

    skyline->next = NULL;
    HARP_skyline_t *head = (HARP_skyline_t *)malloc(sizeof(HARP_skyline_t));
    head->next = skyline;

    printf("HARP_child_t #%d's logical sub-partition {%d, %d, %d, %d}\n", HARP_children[0].id, HARP_children[0].sp_log[0].ts_start, HARP_children[0].sp_log[0].ts_end,
           HARP_children[0].sp_log[0].ch_start, HARP_children[0].sp_log[0].ch_end);

    int cnt = 0;
    while (cnt < 5)
    {
        HARP_skyline_t *tmp = head->next;
        while (skyline != NULL)
        {
            if (skyline->height < tmp->height)
            {
                tmp = skyline;
            }
            skyline = skyline->next;
        }
        skyline = tmp;

        uint8_t hasFit = 0;
        for (int i = 1; i < 6; i++)
        {
            if (HARP_children[i].sp_log[layer].ts_start == 0 && HARP_children[i].sp_log[layer].ts_end == 0 &&
                skyline->width >= HARP_children[i].iface[layer].ts)
            {
                cnt++;
                hasFit = 1;
                HARP_children[i].sp_log[layer].ts_start = skyline->start;
                HARP_children[i].sp_log[layer].ts_end = skyline->start + HARP_children[i].iface[layer].ts;
                HARP_children[i].sp_log[layer].ch_start = skyline->height;
                HARP_children[i].sp_log[layer].ch_end = skyline->height + HARP_children[i].iface[layer].ch;

                printf("HARP_child_t #%d's logical sub-partition {%d, %d, %d, %d}\n", HARP_children[i].id, HARP_children[i].sp_log[0].ts_start, HARP_children[i].sp_log[0].ts_end,
                       HARP_children[i].sp_log[0].ch_start, HARP_children[i].sp_log[0].ch_end);

                if (skyline->width > HARP_children[i].iface[layer].ts)
                {
                    // the remaining part
                    HARP_skyline_t *new_skyline = (HARP_skyline_t *)malloc(sizeof(HARP_skyline_t));
                    new_skyline->start = skyline->start + HARP_children[i].iface[layer].ts;
                    new_skyline->end = skyline->end;
                    new_skyline->width = skyline->width - HARP_children[i].iface[layer].ts;
                    new_skyline->height = skyline->height;
                    new_skyline->prev = skyline;
                    new_skyline->next = skyline->next;

                    // the used part
                    skyline->end = skyline->start + HARP_children[i].iface[layer].ts;
                    skyline->width = HARP_children[i].iface[layer].ts;
                    skyline->height += HARP_children[i].iface[layer].ch;
                    skyline->next = new_skyline;
                }
                else
                {
                    skyline->height += HARP_children[i].iface[layer].ch;
                }
                break;
            }
        }

        // wasted area
        if (!hasFit)
        {
            skyline->prev->end = skyline->end;
            skyline->prev->width += skyline->width;
            skyline->prev->next = skyline->next;
            if (skyline->next != NULL)
                skyline->next->prev = skyline->prev;
            skyline = skyline->prev;
        }

        // merge
        HARP_skyline_t *ss = head->next;
        while (ss != NULL)
        {
            if (ss->width == 0)
            {
                ss->prev = ss->next;
                ss = ss->prev;
            }
            if (ss->next != NULL)
            {
                if (ss->height == ss->next->height)
                {
                    ss->width += ss->next->width;
                    ss->end = ss->next->end;
                    ss->next = ss->next->next;
                    if (ss->next != NULL)
                        ss->next->prev = ss;
                }
            }
            ss = ss->next;
        }
    }
    HARP_skyline_t *s = head->next;
    while (s != NULL)
    {
        if (height < s->height)
            height = s->height;
        HARP_skyline_t *next = s->next;
        free(s);
        s = next;
    }
    printf("Enclosing rectangle: {%d, %d}\n", width, height);

    if (height > MAX_CHANNEL)
    {
        printf("[!] Exceed channel limit (%d), rotate the strip\n", MAX_CHANNEL);
        width = MAX_CHANNEL;
        height = 0;

        for (int i = 0; i < 6; i++)
        {
            HARP_children[i].sp_log[layer].ts_start = 0;
            HARP_children[i].sp_log[layer].ts_end = 0;
            HARP_children[i].sp_log[layer].ch_start = 0;
            HARP_children[i].sp_log[layer].ch_end = 0;
        }
        qsort(HARP_children, 6, sizeof(HARP_child_t), sortChildrenByIfaceCh);


        HARP_skyline_t *skyline = (HARP_skyline_t *)malloc(sizeof(HARP_skyline_t));
        skyline->start = 0;
        skyline->end = width;
        skyline->width = width;
        skyline->height = 0;

        skyline->next = NULL;
        HARP_skyline_t *head = (HARP_skyline_t *)malloc(sizeof(HARP_skyline_t));
        head->next = skyline;
        // skyline->prev = head;

        int cnt = 0;
        while (cnt < 6)
        {
            HARP_skyline_t *tmp = head->next;
            while (skyline != NULL)
            {
                if (skyline->height < tmp->height)
                {
                    tmp = skyline;
                }
                skyline = skyline->next;
            }
            skyline = tmp;

            uint8_t hasFit = 0;
            for (int i = 0; i < 6; i++)
            {
                if (HARP_children[i].sp_log[layer].ts_start == 0 && HARP_children[i].sp_log[layer].ts_end == 0 &&
                    skyline->width >= HARP_children[i].iface[layer].ch)
                {
                    cnt++;
                    hasFit = 1;

                    HARP_children[i].sp_log[layer].ts_start = skyline->height;
                    HARP_children[i].sp_log[layer].ts_end = skyline->height + HARP_children[i].iface[layer].ts;
                    HARP_children[i].sp_log[layer].ch_start = skyline->start;
                    HARP_children[i].sp_log[layer].ch_end = skyline->start + HARP_children[i].iface[layer].ch;

                    printf("HARP_child_t #%d's logical sub-partition {%d, %d, %d, %d}\n", HARP_children[i].id, HARP_children[i].sp_log[0].ts_start, HARP_children[i].sp_log[0].ts_end,
                           HARP_children[i].sp_log[0].ch_start, HARP_children[i].sp_log[0].ch_end);

                    if (skyline->width > HARP_children[i].iface[layer].ch)
                    {
                        // the remaining part
                        HARP_skyline_t *new_skyline = (HARP_skyline_t *)malloc(sizeof(HARP_skyline_t));
                        new_skyline->start = skyline->start + HARP_children[i].iface[layer].ch;
                        new_skyline->end = skyline->end;
                        new_skyline->width = skyline->width - HARP_children[i].iface[layer].ch;
                        new_skyline->height = skyline->height;
                        new_skyline->prev = skyline;
                        new_skyline->next = skyline->next;

                        // the used part
                        skyline->end = skyline->start + HARP_children[i].iface[layer].ch;
                        skyline->width = HARP_children[i].iface[layer].ch;
                        skyline->height += HARP_children[i].iface[layer].ts;
                        skyline->next = new_skyline;
                    }
                    else
                    {
                        skyline->height += HARP_children[i].iface[layer].ts;
                    }
                    break;
                }
            }

            // wasted area
            if (!hasFit)
            {
                skyline->prev->end = skyline->end;
                skyline->prev->width += skyline->width;
                skyline->prev->next = skyline->next;
                if (skyline->next != NULL)
                    skyline->next->prev = skyline->prev;
                skyline = skyline->prev;
            }

            // merge
            HARP_skyline_t *ss = head->next;
            while (ss != NULL)
            {
                if (ss->width == 0)
                {
                    ss->prev = ss->next;
                    ss = ss->prev;
                }
                if (ss->next != NULL)
                {
                    if (ss->height == ss->next->height)
                    {
                        ss->width += ss->next->width;
                        ss->end = ss->next->end;
                        ss->next = ss->next->next;
                        if (ss->next != NULL)
                            ss->next->prev = ss;
                    }
                }
                ss = ss->next;
            }
        }
        HARP_skyline_t *s = head->next;
        while (s != NULL)
        {
            if (height < s->height)
                height = s->height;
            HARP_skyline_t *next = s->next;
            free(s);
            s = next;
        }
        printf("Enclosing rectangle: {%d, %d}\n", width, height);
    }

    return 0;
}

int main()
{
    initChildren();
    skylinePacking();
    return 0;
}