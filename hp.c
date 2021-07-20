#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <time.h>

#define MAX_CHANNEL 4

typedef struct
{
    uint8_t id;
    uint8_t iface[5][2];
    uint8_t sp_log[5][4];
    uint8_t sp_phy[5][4];
} Child;

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

int cmpTs(const void *a, const void *b)
{
    return ((uint8_t *)(b))[0] - ((uint8_t *)(a))[0];
}

int cmpCh(const void *a, const void *b)
{
    return ((uint8_t *)(b))[1] - ((uint8_t *)(a))[1];
}

typedef struct __skyline_t
{
    uint8_t start;
    uint8_t end;
    uint8_t width;
    uint8_t height;
    struct __skyline_t *prev;
    struct __skyline_t *next;
} skyline_t;

void printSkyline(skyline_t *s)
{
    printf("\tstart: %d, end: %d, width: %d, height: %d\n",
           s->start, s->end, s->width, s->height);
}

uint8_t width, height;

uint8_t skylinePacking()
{
    width = 0;
    height = 0;
    printf("Best-fit Skyline Packing\n");
    // optimal: {9,5}
    uint8_t rects[6][3] = {{9, 1, 0}, {8, 2, 0}, {1, 1, 0}, {6, 1, 0}, {2, 2, 0}, {2, 1, 0}};
    printf("Input rectangles\n");
    // printRect(rects, 6);

    qsort(rects, 6, 3 * sizeof(uint8_t), cmpTs);
    printf("Sorted by width (slots)\n");
    // printRect(rects, 6);

    width = rects[0][0];

    skyline_t *skyline = (skyline_t *)malloc(sizeof(skyline_t));
    skyline->start = 0;
    skyline->end = width;
    skyline->width = width;
    skyline->height = rects[0][1];
    // printf("Initialize skyline\n");
    // printSkyline(skyline);

    skyline->next = NULL;
    skyline_t *head = (skyline_t *)malloc(sizeof(skyline_t));
    head->next = skyline;
    // skyline->prev = head;

    int cnt = 0;
    while (cnt < 5)
    {
        skyline_t *tmp = head->next;
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
                // printf("place [%d, %d]\n", rects[i][0], rects[i][1]);
                cnt++;
                hasFit = 1;
                rects[i][2] = 1;
                if (skyline->width > rects[i][0])
                {
                    // the remaining part
                    skyline_t *new_skyline = (skyline_t *)malloc(sizeof(skyline_t));
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
        skyline_t *ss = head->next;
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
    skyline_t *s = head->next;
    while (s != NULL)
    {
        if (height < s->height)
            height = s->height;
        skyline_t *next = s->next;
        free(s);
        s = next;
    }
    printf("Enclosing rectangle: {%d, %d}\n", width, height);

    if (height > MAX_CHANNEL)
    {
        printf("Exceed channel limit (%d), rotate the strip\n", MAX_CHANNEL);
        width = MAX_CHANNEL;
        height = 0;

        for (int i = 0; i < 6; i++)
        {
            rects[i][2] = 0;
        }

        qsort(rects, 6, 3 * sizeof(uint8_t), cmpCh);
        // printf("Sorted by width (channels)\n");
        // printRect(rects, 6);

        skyline_t *skyline = (skyline_t *)malloc(sizeof(skyline_t));
        skyline->start = 0;
        skyline->end = width;
        skyline->width = width;
        skyline->height = 0;

        skyline->next = NULL;
        skyline_t *head = (skyline_t *)malloc(sizeof(skyline_t));
        head->next = skyline;
        // skyline->prev = head;

        int cnt = 0;
        while (cnt < 6)
        {
            skyline_t *tmp = head->next;
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
                if (rects[i][2] == 0 && skyline->width >= rects[i][1])
                {
                    // printf("place [%d, %d]\n", rects[i][0], rects[i][1]);
                    cnt++;
                    hasFit = 1;
                    rects[i][2] = 1;
                    if (skyline->width > rects[i][1])
                    {
                        // the remaining part
                        skyline_t *new_skyline = (skyline_t *)malloc(sizeof(skyline_t));
                        new_skyline->start = skyline->start + rects[i][1];
                        new_skyline->end = skyline->end;
                        new_skyline->width = skyline->width - rects[i][1];
                        new_skyline->height = skyline->height;
                        new_skyline->prev = skyline;
                        new_skyline->next = skyline->next;

                        // the used part
                        skyline->end = skyline->start + rects[i][1];
                        skyline->width = rects[i][1];
                        skyline->height += rects[i][0];
                        skyline->next = new_skyline;
                    }
                    else
                    {
                        skyline->height += rects[i][0];
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
            skyline_t *ss = head->next;
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
        skyline_t *s = head->next;
        while (s != NULL)
        {
            if (height < s->height)
                height = s->height;
            skyline_t *next = s->next;
            free(s);
            s = next;
        }
        printf("Enclosing rectangle: {%d, %d}\n", width, height);
    }

    return 0;
}

int main()
{
    clock_t begin, end;
    double cost;

    begin = clock();
    skylinePacking();
    end = clock();
    cost = (double)(end - begin);
    printf("time cost is: %f us\n", cost);

    return 0;
}