int fib(int n)
{
    if (n < 0)
    {
        return -1;
    }
    if (n == 0 || n == 1)
    {
        return 1;
    }

    return fib(n - 1) + fib(n - 2);
}

int main()
{
    int i = 0;
    while (i <= 10)
    {
        print_int(fib(i));
        print_newline();
        i = i + 1;
    }
    return 0;
}