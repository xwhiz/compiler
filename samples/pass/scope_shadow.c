int main() {
    int x = 10;

    {
        int x = 3;
        print_int(x);
        print_newline();
    }

    print_int(x);
    print_newline();
    return 0;
}
