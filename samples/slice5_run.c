int main() {
    int i = 0;
    int sum = 0;

    while (i < 5) {
        if (i == 3) {
            print_int(99);
            print_newline();
        }

        sum = sum + i;
        i = i + 1;
    }

    print_int(sum);
    print_newline();
    return sum;
}
