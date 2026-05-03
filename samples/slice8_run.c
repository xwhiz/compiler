int main() {
    char s[6] = "hello";
    int a[3];
    a[0] = 4;
    a[1] = 5;
    a[2] = a[0] + a[1];
    print_str(s);
    print_newline();
    print_int(a[2]);
    print_newline();
    return a[2];
}
