#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#define MAX_CHANNEL 15

#define HARP_SEND_DELAY 200 // delay between each harp packet, unit: slot

#define MAX_CHILDREN_NUM 10

#define MAX_HOP 5 // start from 0
#define TRAFFIC_UNIT 1

#define HARP_MAILBOX_DUMMY 0x01
#define HARP_MAILBOX_SEND_INIT 0x02
#define HARP_MAILBOX_SEND_IFACE 0x03
#define HARP_MAILBOX_SEND_SP 0x04
#define HARP_MAILBOX_SEND_SCH 0x05

static uint8_t cur_sort_layer;

typedef struct
{
    uint16_t ts;
    uint8_t ch;
} HARP_interface_t;

typedef struct
{
    uint16_t ts_start;
    uint16_t ts_end;
    uint8_t ch_start;
    uint8_t ch_end;
} HARP_subpartition_t;

typedef struct
{
    uint8_t id;
    uint8_t traffic;
    HARP_interface_t iface[MAX_HOP + 1];
    HARP_subpartition_t sp_rel[MAX_HOP + 1];
    HARP_subpartition_t sp_abs[MAX_HOP + 1];
} HARP_child_t;

typedef struct
{
    uint8_t id;
    uint8_t parent;
    uint8_t layer;
    uint8_t children_cnt;
    uint8_t recv_iface_cnt;
    uint16_t partition_center; // to copy an uplink cell to a downlink cell
    uint16_t partition_up_end; // uplink partition end (right edge)
    uint8_t traffic_fwd;
    uint8_t adjustingNodes[2];
    uint8_t adjustingLayer;
    uint8_t relocatedCnt;
    HARP_interface_t iface[MAX_HOP + 1];
    HARP_subpartition_t sp_abs[MAX_HOP + 1];
} HARP_self_t;

typedef struct __HARP_skyline_t
{
    uint16_t start;
    uint16_t end;
    uint16_t width;
    uint16_t height;
    struct __HARP_skyline_t *prev;
    struct __HARP_skyline_t *next;
} HARP_skyline_t;

HARP_child_t HARP_children[MAX_CHILDREN_NUM];
HARP_self_t HARP_self;

uint8_t interfaceComposition();
uint8_t changeInterface();
uint8_t subpartitionAdjustment();
uint8_t findIdleRectangularAreas(uint8_t idleRectangles[16][4]);

int sortChildrenByIfaceTs(const void *a, const void *b)
{
    return ((HARP_child_t *)b)->iface[cur_sort_layer].ts - ((HARP_child_t *)a)->iface[cur_sort_layer].ts;
}

int sortChildrenByIfaceTsInc(const void *a, const void *b)
{
    return ((HARP_child_t *)a)->iface[cur_sort_layer].ts - ((HARP_child_t *)b)->iface[cur_sort_layer].ts;
}

int sortChildrenByIfaceCh(const void *a, const void *b)
{
    return ((HARP_child_t *)b)->iface[cur_sort_layer].ch - ((HARP_child_t *)a)->iface[cur_sort_layer].ch;
}

int init()
{
    memset(&HARP_self, 0, sizeof(HARP_self));
    memset(&HARP_children[0], 0, sizeof(HARP_child_t) * MAX_CHILDREN_NUM);

    HARP_self.id = 1;
    HARP_self.children_cnt = 7;

    uint8_t childrenIface[7][2] = {{3, 1}, {5, 2}, {3, 1}, {4, 1}, {1, 1}, {4, 2}, {6, 2}};

    for (uint8_t i = 0; i < 7; i++)
    {
        HARP_children[i].id = i + 2;
        HARP_children[i].iface[1].ts = childrenIface[i][0];
        HARP_children[i].iface[1].ch = childrenIface[i][1];
    }
    interfaceComposition();
    for (uint8_t i = 0; i < 7; i++)
    {
        printf("#%d iface: [%d,%d], rel_sp: [%d,%d,%d,%d]\n", HARP_children[i].id, HARP_children[i].iface[1].ts,
               HARP_children[i].iface[1].ch, HARP_children[i].sp_rel[1].ts_start, HARP_children[i].sp_rel[1].ts_end,
               HARP_children[i].sp_rel[1].ch_start, HARP_children[i].sp_rel[1].ch_end);
    }
    // uint8_t childrenIface[7][2] = {{3, 1}, {5, 2}, {3, 1}, {4, 1}, {1, 1}, {4, 2}, {6, 2}};

    return 0;
}

int main()
{
    init();
    changeInterface();
    while (HARP_self.adjustingNodes[0] != 0)
    {
        subpartitionAdjustment();
    }
    // findIdleRectangularAreas();
    return 1;
}

uint8_t changeInterface()
{
    for (uint8_t i = 0; i < MAX_CHILDREN_NUM; i++)
    {
        if (HARP_children[i].id == 3)
        {
            HARP_children[i].iface[1].ts = 6;
            HARP_children[i].iface[1].ch = 2;
        }
    }
    HARP_self.adjustingNodes[0] = 3;
    HARP_self.adjustingLayer = 1;
    return 1;
}

uint8_t subpartitionAdjustment()
{
    HARP_interface_t adjIface;
    for (uint8_t i = 0; i < MAX_CHILDREN_NUM; i++)
    {
        if (HARP_children[i].id == HARP_self.adjustingNodes[0])
        {
            adjIface.ts = HARP_children[i].iface[HARP_self.adjustingLayer].ts;
            adjIface.ch = HARP_children[i].iface[HARP_self.adjustingLayer].ch;
            break;
        }
    }

    printf("to be adjusted iface: I_(%d,%d)=(%d, %d)\n",
           HARP_self.adjustingNodes[0], HARP_self.adjustingLayer, adjIface.ts, adjIface.ch);

    uint8_t idleRectangles[16][4] = {};

    uint8_t rectCnt = findIdleRectangularAreas(idleRectangles);
    printf("idle rectangles: ");
    for (uint8_t i = 0; i < rectCnt; i++)
    {
        printf("[%d,%d,%d,%d],", idleRectangles[i][0], idleRectangles[i][1],
               idleRectangles[i][2], idleRectangles[i][3]);
    }
    printf("\n");

    uint8_t found = 0;
    for (uint8_t i = 0; i < rectCnt; i++)
    {
        if ((idleRectangles[i][1] - idleRectangles[i][0]) >= adjIface.ts &&
            (idleRectangles[i][3] - idleRectangles[i][2]) >= adjIface.ch)
        {
            found = 1;
            printf("fit place found, place #%d's iface@l%d (%d, %d) to [%d,%d,%d,%d]\n",
                   HARP_self.adjustingNodes[0], HARP_self.adjustingLayer, adjIface.ts, adjIface.ch,
                   idleRectangles[i][0], idleRectangles[i][1], idleRectangles[i][2], idleRectangles[i][3]);
            // update rel_sp
            for (uint8_t c = 0; c <= HARP_self.children_cnt; c++)
            {
                if (HARP_self.adjustingNodes[0] == HARP_children[c].id)
                {
                    HARP_children[c].sp_rel[HARP_self.adjustingLayer].ts_start = idleRectangles[i][0];
                    HARP_children[c].sp_rel[HARP_self.adjustingLayer].ts_end = idleRectangles[i][0] + adjIface.ts;
                    HARP_children[c].sp_rel[HARP_self.adjustingLayer].ch_start = idleRectangles[i][2];
                    HARP_children[c].sp_rel[HARP_self.adjustingLayer].ch_end = idleRectangles[i][2] + adjIface.ch;
                    break;
                }
            }
            HARP_self.adjustingNodes[0] = 0;
            return 1;
        }
    }
    printf("not found, try to relocate another sp first\n");
    // try if relocating another sp can give a feasible placement
    if (!found)
    {
        cur_sort_layer = HARP_self.adjustingLayer;
        qsort(HARP_children, MAX_CHILDREN_NUM, sizeof(HARP_child_t), sortChildrenByIfaceTsInc);

        for (uint8_t c = 0; c < MAX_CHILDREN_NUM; c++)
        {
            if (HARP_children[c].iface[HARP_self.adjustingLayer].ts == 0 || HARP_children[c].id == HARP_self.adjustingNodes[0])
                continue;
            printf("[%d,#%d]\n", c, HARP_children[c].id);
            HARP_self.adjustingNodes[1] = HARP_children[c].id;

            uint8_t rectCnt = findIdleRectangularAreas(idleRectangles);
            printf("%d idle rectangles: ", c);
            for (uint8_t i = 0; i < rectCnt; i++)
            {
                printf("[%d,%d,%d,%d],", idleRectangles[i][0], idleRectangles[i][1],
                       idleRectangles[i][2], idleRectangles[i][3]);
            }
            printf("\n");

            uint8_t found = 0;
            for (uint8_t i = 0; i < rectCnt; i++)
            {
                if ((idleRectangles[i][1] - idleRectangles[i][0]) >= adjIface.ts &&
                    (idleRectangles[i][3] - idleRectangles[i][2]) >= adjIface.ch)
                {
                    found = 1;
                    printf("fit place found by moving #%d away, place #%d's iface@l%d (%d, %d) to [%d,%d,%d,%d]\n", HARP_children[c].id,
                           HARP_self.adjustingNodes[0], HARP_self.adjustingLayer, adjIface.ts, adjIface.ch,
                           idleRectangles[i][0], idleRectangles[i][1], idleRectangles[i][2], idleRectangles[i][3]);
                    // update rel_sp
                    for (uint8_t c = 0; c <= HARP_self.children_cnt; c++)
                    {
                        if (HARP_self.adjustingNodes[0] == HARP_children[c].id)
                        {
                            HARP_children[c].sp_rel[HARP_self.adjustingLayer].ts_start = idleRectangles[i][0];
                            HARP_children[c].sp_rel[HARP_self.adjustingLayer].ts_end = idleRectangles[i][0] + adjIface.ts;
                            HARP_children[c].sp_rel[HARP_self.adjustingLayer].ch_start = idleRectangles[i][2];
                            HARP_children[c].sp_rel[HARP_self.adjustingLayer].ch_end = idleRectangles[i][2] + adjIface.ch;
                            break;
                        }
                    }
                    HARP_self.adjustingNodes[0] = HARP_self.adjustingNodes[1];
                    HARP_self.adjustingNodes[1] = 0;
                    return 1;
                }
            }
        }
    }
    return 1;
}

void print_binary(unsigned int number)
{
    if (number >> 1)
    {
        print_binary(number >> 1);
    }
    putc((number & 1) ? '1' : '0', stdout);
}

uint8_t findIdleRectangularAreas(uint8_t idleRectangles[16][4])
{
    uint32_t rectBitmap[MAX_CHANNEL] = {};
    uint8_t rectCnt = 0;
    uint8_t layer = HARP_self.adjustingLayer;

    for (uint8_t i = 0; i < MAX_CHILDREN_NUM; i++)
    {
        if (HARP_children[i].iface[layer].ts == 0 ||
            HARP_children[i].id == HARP_self.adjustingNodes[0] || HARP_children[i].id == HARP_self.adjustingNodes[1])
            continue;
        printf("child id: %d\n", HARP_children[i].id);
        for (uint8_t y = HARP_children[i].sp_rel[layer].ch_start; y < HARP_children[i].sp_rel[layer].ch_end; y++)
        {
            for (uint8_t x = HARP_children[i].sp_rel[layer].ts_start; x < HARP_children[i].sp_rel[layer].ts_end; x++)
            {
                rectBitmap[y] |= 0x80000000 >> x;
            }
        }
    }

    for (int8_t i = HARP_self.iface[layer].ch - 1; i >= 0; i--)
    {
        print_binary(rectBitmap[i]);
        printf("\n");
    }

    for (uint8_t yCur = 0; yCur < HARP_self.iface[layer].ch; yCur++)
    {
        for (uint8_t xCur = 0; xCur < HARP_self.iface[layer].ts; xCur++)
        {
            if ((rectBitmap[yCur] << xCur & 0x80000000) == 0)
            {
                uint8_t xStart = xCur;
                uint8_t xEnd = xCur;
                uint8_t yStart = yCur;
                uint8_t yEnd = yCur;
                for (uint8_t yy = yCur; yy < HARP_self.iface[layer].ch; yy++)
                {
                    if ((rectBitmap[yy] << xCur & 0x80000000) != 0)
                    {
                        yEnd = yy;
                        break;
                    }
                    if (yy == HARP_self.iface[layer].ch - 1)
                    {
                        yEnd = yy + 1;
                    }
                }
                for (uint8_t xx = xCur; xx < HARP_self.iface[layer].ts; xx++)
                {
                    if ((rectBitmap[yCur] << xx & 0x80000000) != 0)
                    {
                        xEnd = xx;
                        break;
                    }
                    if (xx == HARP_self.iface[layer].ts - 1)
                    {
                        xEnd = xx + 1;
                    }
                }

                uint8_t duplicated = 0;
                for (uint8_t i = 0; i < rectCnt; i++)
                {
                    if (xStart >= idleRectangles[i][0] && xEnd <= idleRectangles[i][1] &&
                        yStart >= idleRectangles[i][2] && yEnd <= idleRectangles[i][3])
                    {
                        duplicated = 1;
                        break;
                    }
                }
                if (!duplicated)
                {
                    // printf("rectCnt: %d\n",rectCnt);
                    idleRectangles[rectCnt][0] = xStart;
                    idleRectangles[rectCnt][1] = xEnd;
                    idleRectangles[rectCnt][2] = yStart;
                    idleRectangles[rectCnt][3] = yEnd;
                    rectCnt++;
                }
            }
        }
    }
    return rectCnt;
}

uint8_t interfaceComposition()
{
    printf("Composited interface: \n");
    for (uint8_t layer = 0; layer <= MAX_HOP; layer++)
    {
        uint16_t width = 0;
        uint8_t height = 0;

        // for qsort
        cur_sort_layer = layer;

        uint8_t children_cnt = 0;
        for (uint8_t i = 0; i < HARP_self.children_cnt; i++)
            if (HARP_children[i].iface[layer].ts != 0)
                children_cnt++;
        if (children_cnt == 0)
            continue;

        qsort(HARP_children, MAX_CHILDREN_NUM, sizeof(HARP_child_t), sortChildrenByIfaceTs);
        HARP_children[0].sp_rel[layer].ts_start = 0;
        HARP_children[0].sp_rel[layer].ts_end = HARP_children[0].iface[layer].ts;
        HARP_children[0].sp_rel[layer].ch_start = 0;
        HARP_children[0].sp_rel[layer].ch_end = 0 + HARP_children[0].iface[layer].ch;
        width = HARP_children[0].iface[layer].ts;
        HARP_skyline_t *skyline = (HARP_skyline_t *)malloc(sizeof(HARP_skyline_t));
        skyline->start = 0;
        skyline->end = width;
        skyline->width = width;
        skyline->height = HARP_children[0].iface[layer].ch;
        skyline->next = NULL;
        HARP_skyline_t *head = (HARP_skyline_t *)malloc(sizeof(HARP_skyline_t));
        head->next = skyline;
        int cnt = 0;
        while (cnt < children_cnt - 1)
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
            for (int i = 1; i < children_cnt; i++)
            {
                if (HARP_children[i].sp_rel[layer].ts_start == 0 && HARP_children[i].sp_rel[layer].ts_end == 0 &&
                    skyline->width >= HARP_children[i].iface[layer].ts)
                {
                    cnt++;
                    hasFit = 1;
                    HARP_children[i].sp_rel[layer].ts_start = skyline->start;
                    HARP_children[i].sp_rel[layer].ts_end = skyline->start + HARP_children[i].iface[layer].ts;
                    HARP_children[i].sp_rel[layer].ch_start = skyline->height;
                    HARP_children[i].sp_rel[layer].ch_end = skyline->height + HARP_children[i].iface[layer].ch;

                    // printf("HARP_child_t #%d's rel_sp@l%d {%d, %d, %d, %d}\n", HARP_children[i].id, layer, HARP_children[i].sp_rel[layer].ts_start, HARP_children[i].sp_rel[layer].ts_end,
                    //        HARP_children[i].sp_rel[layer].ch_start, HARP_children[i].sp_rel[layer].ch_end);

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

        HARP_self.iface[layer].ts = width;
        HARP_self.iface[layer].ch = height;

        printf("l%d-{%d, %d}, ", layer, HARP_self.iface[layer].ts, HARP_self.iface[layer].ch);
    }
    printf("\n");
    return 0;
}