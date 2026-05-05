int main()
{
    // MAKE SURE THIS IS ODD TO PRINT A CORRECT ONE
    int MAX_COUNT = 11;
    int i = 0;
    while (i < MAX_COUNT)
    {
        int j = 0;
        while (j < (MAX_COUNT - i) / 2)
        {
            print_char(' ');
            j = j + 1;
        }

        j = 0;
        while (j <= i)
        {
            print_char('*');
            j = j + 1;
        }
        print_newline();
        i = i + 2;
    }
}
