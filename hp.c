#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>

struct Child
{
    uint8_t id;
    uint8_t iface[5][2];
    uint8_t sp_log[5][4];
    uint8_t sp_phy[5][4];
};

void printRect(uint8_t rects[][3], uint8_t m)
{
    for (uint8_t i = 0; i < m; i++)
    {
        printf("\t{");
        for (uint8_t j = 0; j < 2; j++)
        {
            printf("%d,", rects[i][j]);
        }
        printf("\b}, ");
    }
    printf("\n");
}

int cmp(const void *a, const void *b)
{
    return (*(uint8_t *)(b) - *(uint8_t *)(a));
}

struct skyline_t
{
    uint8_t start;
    uint8_t end;
    uint8_t width;
    uint8_t height;
    struct skyline_t *prev;
    struct skyline_t *next;
};

void printSkyline(struct skyline_t *s)
{
    printf("\tstart: %d, end: %d, width: %d, height: %d\n",
           s->start, s->end, s->width, s->height);
}

uint8_t skylinePacking()
{
    printf("Best-fit Skyline Packing\n");

    uint8_t width = 0, height = 0;

    // optimal: {9,5}
    uint8_t rects[6][3] = {
        {9, 1, 0}, {8, 2, 0}, {1, 1, 0}, {6, 1, 0}, {2, 2, 0}, {2, 1, 0}};
    printf("Input rectangles\n");
    printRect(rects, 6);

    qsort(rects, 6, 3 * sizeof(uint8_t), cmp);
    printf("Sorted by width\n");
    printRect(rects, 6);

    width = rects[0][0];

    struct skyline_t *skyline = (struct skyline_t *)malloc(sizeof(struct skyline_t));
    skyline->start = 0;
    skyline->end = width;
    skyline->width = width;
    skyline->height = rects[0][1];
    printf("Initialize skyline\n");
    printSkyline(skyline);

    skyline->next = NULL;
    struct skyline_t *head = (struct skyline_t *)malloc(sizeof(struct skyline_t));
    head->next = skyline;
    // skyline->prev = head;

    int cnt = 0;
    while (cnt < 5)
    {
        struct skyline_t *tmp = head->next;
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
            if (rects[i][2] == 0 && skyline->width >= rects[i][0])
            {
                printf("place [%d, %d]\n", rects[i][0], rects[i][1]);
                cnt++;
                hasFit = 1;
                rects[i][2] = 1;
                if (skyline->width > rects[i][0])
                {
                    // the remaining part
                    struct skyline_t *new_skyline = (struct skyline_t *)malloc(sizeof(struct skyline_t));
                    new_skyline->start = skyline->start + rects[i][0];
                    new_skyline->end = skyline->end;
                    new_skyline->width = skyline->width - rects[i][0];
                    new_skyline->height = skyline->height;
                    new_skyline->prev = skyline;
                    new_skyline->next = skyline->next;

                    // the used part
                    skyline->end = skyline->start + rects[i][0];
                    skyline->width = rects[i][0];
                    skyline->height += rects[i][1];
                    skyline->next = new_skyline;
                }
                else
                {
                    skyline->height += rects[i][1];
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
        struct skyline_t *ss = head->next;
        while (ss != NULL)
        {
            if (ss->width ==0)
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
        // struct skyline_t *s = head->next;
        // while (s != NULL)
        // {
        //     printSkyline(s);
        //     s = s->next;
        // }
    }
    return 0;
}

int main()
{
    skylinePacking();
    return 0;
}