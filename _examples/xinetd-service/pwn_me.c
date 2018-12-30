#include <stdio.h>
#include <unistd.h>

int sample()
{	FILE *ptr_file;
	char buf[100];

	ptr_file = fopen("flag.txt","r");
	if (!ptr_file)
		return 1;

	while (fgets(buf,100, ptr_file)!=NULL)
		fprintf(stderr, "%s",buf);
	fclose(ptr_file);
	return 0;
}

void test()
{	char input[50];
	gets(input);
	sleep(1);
	fprintf(stderr, "ECHO: %s\n",input); 
}

int main()
{	test();
	return 0;
}
