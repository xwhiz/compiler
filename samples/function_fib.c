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
    int 1fibs [10];
    int i = 0;
    while (i < 10)
    {
        int result = fib(i);
        fibs[i] = result;
        i = i + 1;
    }

    i = 0;
    while (i < 10)
    {

        print_int(fibs[i]);
        print_newline();
        i = i + 1;
    }
    return 0;
}